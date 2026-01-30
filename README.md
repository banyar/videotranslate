# Video - YouTube to Burmese Video Translator

A Go CLI tool that downloads YouTube videos and translates them to Burmese with subtitles and audio.

## Features

- Download YouTube videos
- Speech-to-text transcription (Whisper)
- English to Burmese translation
- Burmese text-to-speech (Edge TTS)
- Burn Burmese subtitles into video
- Live English to Burmese translation mode

## Prerequisites

- Go 1.23+
- Python 3.12+ with venv
- ffmpeg
- arecord (for live mode, Linux only)

## Installation

### 1. Clone and setup Go dependencies

```bash
git clone https://github.com/banyar-sithu/video.git
cd video
go mod download
```

### 2. Setup Python virtual environment

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install openai-whisper edge-tts deep-translator
```

### 3. Install system dependencies

```bash
# Ubuntu/Debian
sudo apt install ffmpeg

# For live mode
sudo apt install alsa-utils
```

### 4. Create environment file

```bash
cp .env.example .env
# Edit .env and set DOWNLOAD_YOUTUBE_URL
```

`.env` file:
```
DOWNLOAD_YOUTUBE_URL=https://www.youtube.com/watch?v=VIDEO_ID
```

## Usage

### Build the CLI

```bash
go build -o video .
```

### Commands

#### Convert YouTube Video to Burmese

Downloads a YouTube video and creates a Burmese version with translated subtitles and audio.

```bash
./video burmese
```

Output files will be saved to `ToBurmeseVideoOutput/<video_title>/`:
- `<video_title>.mp4` - Original video
- `<video_title>_english.txt` - English transcription
- `<video_title>_english.srt` - English subtitles
- `<video_title>_burmese.txt` - Burmese translation
- `<video_title>_burmese.srt` - Burmese subtitles
- `<video_title>_burmese.mp3` - Burmese audio
- `<video_title>_with_subs.mp4` - Video with burned subtitles
- `<video_title>_burmese.mp4` - Final video with Burmese audio and subtitles

#### Live Translation Mode

Real-time English to Burmese speech translation.

```bash
./video live
```

Press `Ctrl+C` to stop.

#### Check Version

```bash
./video version
```

## Running with Go

You can also run directly without building:

```bash
go run . burmese
go run . live
go run . version
```
