#!/usr/bin/env python3
"""Yerel transkripsiyon — faster-whisper (offline, anahtar gerekmez).

Kullanım:
    python whisper_transcribe.py --audio /yol/ses.mp3 --language tr --model medium

Çıktı (stdout, tek JSON nesnesi):
    {
      "language": "tr",
      "duration": 496.0,
      "text": "…",
      "segments": [
        {"start": 0.0, "end": 4.2, "text": "…", "words": [{"w": "…", "s": 0.0, "e": 0.4}]}
      ]
    }

İlerleme stderr'e `PROGRESS <yüzde>` satırlarıyla yazılır; Go tarafı bunu
kuyruk ilerlemesine çevirir. Ses hiçbir yere gönderilmez: gizli röportaj
kayıtları için varsayılan sağlayıcı bu olabilir.

Hatalar stderr'e `ERROR <mesaj>` olarak yazılır ve çıkış kodu 1 olur.
"""
import argparse
import json
import os
import sys


def emit_progress(pct: int) -> None:
    pct = max(0, min(99, int(pct)))
    sys.stderr.write(f"PROGRESS {pct}\n")
    sys.stderr.flush()


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--audio", required=True)
    ap.add_argument("--language", default="tr")
    ap.add_argument("--model", default="medium")
    ap.add_argument("--device", default="cpu")
    ap.add_argument("--compute-type", default="int8")
    ap.add_argument("--beam-size", type=int, default=5)
    ap.add_argument("--threads", type=int, default=0)
    ap.add_argument("--output", default="")
    args = ap.parse_args()

    if not os.path.isfile(args.audio):
        sys.stderr.write(f"ERROR ses dosyası bulunamadı: {args.audio}\n")
        return 1

    try:
        from faster_whisper import WhisperModel
    except Exception as exc:  # noqa: BLE001
        sys.stderr.write(f"ERROR faster-whisper kurulu değil: {exc}\n")
        return 1

    cpu_threads = args.threads or max(1, (os.cpu_count() or 4))
    try:
        model = WhisperModel(
            args.model,
            device=args.device,
            compute_type=args.compute_type,
            cpu_threads=cpu_threads,
        )
    except Exception as exc:  # noqa: BLE001
        sys.stderr.write(f"ERROR model yüklenemedi ({args.model}): {exc}\n")
        return 1

    emit_progress(2)

    try:
        segments_iter, info = model.transcribe(
            args.audio,
            language=args.language or None,
            beam_size=args.beam_size,
            vad_filter=True,
            vad_parameters={"min_silence_duration_ms": 500},
            word_timestamps=True,
            condition_on_previous_text=False,
        )
    except Exception as exc:  # noqa: BLE001
        sys.stderr.write(f"ERROR transkripsiyon başlatılamadı: {exc}\n")
        return 1

    duration = float(getattr(info, "duration", 0.0) or 0.0)
    segments = []
    text_parts = []
    for seg in segments_iter:
        words = []
        for w in (seg.words or []):
            token = (w.word or "").strip()
            if not token:
                continue
            words.append({"w": token, "s": round(float(w.start), 3), "e": round(float(w.end), 3)})
        text = (seg.text or "").strip()
            text_parts.append(text)
        segments.append({
            "start": round(float(seg.start), 3),
            "end": round(float(seg.end), 3),
            "text": text,
            "words": words,
        })
        if duration > 0:
            emit_progress(2 + (seg.end / duration) * 96)

    emit_progress(99)

    payload = {
        "language": getattr(info, "language", args.language) or args.language,
        "duration": round(duration, 3),
        "text": " ".join(text_parts).strip(),
        "segments": segments,
    }
    raw = json.dumps(payload, ensure_ascii=False)
    if args.output:
        with open(args.output, "w", encoding="utf-8") as fh:
            fh.write(raw)
    else:
        sys.stdout.write(raw)
    return 0


if __name__ == "__main__":
    sys.exit(main())
