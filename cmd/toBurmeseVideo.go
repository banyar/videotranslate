package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kkdai/youtube/v2"
	"github.com/spf13/cobra"
)

var name string

var toBurmeseCmd = &cobra.Command{
	Use:   "burmese",
	Short: "Video download from youtube and to change burmese language video",
	Long:  "Print a message. Use --name to specify who to .",
	Run: func(cmd *cobra.Command, args []string) {
		video()
	},
}

func init() {
	toBurmeseCmd.Flags().StringVarP(&name, "name", "n", "World", "name of the person to greet")
	rootCmd.AddCommand(toBurmeseCmd)
}

func video() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("❌ Failed to load .env file:", err)
		return
	}

	// Get YouTube URL from environment
	youtubeURL := os.Getenv("DOWNLOAD_YOUTUBE_URL")
	if youtubeURL == "" {
		fmt.Println("❌ DOWNLOAD_YOUTUBE_URL not set in .env file")
		return
	}

	// Get video info to create output directory based on title
	client := &youtube.Client{}
	videoInfo, err := client.GetVideo(youtubeURL)
	if err != nil {
		fmt.Println("❌ Failed to get video info:", err)
		return
	}

	// Create output directory based on video title
	baseName := sanitizeFileName(videoInfo.Title)
	outputDir := filepath.Join("ToBurmeseVideoOutput", baseName)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Println("❌ Failed to create output directory:", err)
		return
	}
	fmt.Printf("📁 Output directory: %s\n", outputDir)

	// Step 1: Download video
	err = videoDownloadProcess(youtubeURL, videoInfo, outputDir, baseName)
	if err != nil {
		fmt.Println("❌ Download Error:", err)
		return
	}

	// File names based on video name (all inside baseFile folder)
	videoFile := filepath.Join(outputDir, baseName+".mp4")
	englishFile := filepath.Join(outputDir, baseName+"_english.txt")
	burmeseFile := filepath.Join(outputDir, baseName+"_burmese.txt")
	burmeseAudio := filepath.Join(outputDir, baseName+"_burmese.mp3")
	outputVideo := filepath.Join(outputDir, baseName+"_burmese.mp4")

	// Step 2: Speech-to-Text (Whisper)
	fmt.Println("\n🎤 Speech-to-Text ဆောင်ရွက်နေသည်...")
	_, err = speechToText(videoFile, englishFile)
	if err != nil {
		fmt.Println("❌ Error:", err)
		return
	}
	fmt.Printf("✅ အင်္ဂလိပ်စာ saved to: %s\n\n", englishFile)

	// Step 3: Translation (English → Burmese)
	fmt.Println("🔤 မြန်မာစာ အဘိဒ္ဒာန ဆောင်ရွက်နေသည်...")
	englishContent, err := os.ReadFile(englishFile)
	if err != nil {
		fmt.Println("❌ Error reading:", err)
		return
	}
	_, err = translateToBurmese(string(englishContent), burmeseFile)
	if err != nil {
		fmt.Println("❌ Error:", err)
		return
	}
	fmt.Printf("✅ မြန်မာစာ saved to: %s\n\n", burmeseFile)

	// Step 4: Text-to-Speech (Burmese)
	fmt.Println("\n🔊 Burmese TTS ဆောင်ရွက်နေသည်...")
	err = textToSpeechBurmese(burmeseFile, burmeseAudio)
	if err != nil {
		fmt.Println("❌ TTS Error:", err)
		return
	}

	// Step 5: Merge audio with video
	fmt.Println("\n🎬 Video နှင့် Audio ပေါင်းစပ်နေသည်...")
	err = mergeAudioWithVideo(videoFile, burmeseAudio, outputVideo)
	if err != nil {
		fmt.Println("❌ Merge Error:", err)
		return
	}

	fmt.Printf("\n🎉 Complete! Final video: %s\n", outputVideo)
}

func videoDownloadProcess(youtubeURL string, videoInfo *youtube.Video, outputDir, baseName string) error {
	// Step 1: YouTube ဒေါင်းလုပ်ခြင်း
	fmt.Println("🎥 YouTube ဒေါင်းလုပ်နေသည်...")
	err := downloadYouTube(videoInfo, outputDir, baseName)
	if err != nil {
		return err
	}
	fmt.Printf("✅ Video saved: %s/%s.mp4\n", outputDir, baseName)
	return nil
}

// YouTube ဒေါင်းလုပ်ခြင်း
func downloadYouTube(video *youtube.Video, outputDir, baseName string) error {
	client := &youtube.Client{}

	fmt.Printf("📹 Title: %s\n", video.Title)

	formats := video.Formats.WithAudioChannels()
	if len(formats) == 0 {
		return fmt.Errorf("audio format မရှိ")
	}

	// ပထမ audio format ရွေးချယ်ခြင်း
	format := &formats[0]
	totalSize := format.ContentLength
	fmt.Printf("📦 Size: %.2f MB\n", float64(totalSize)/(1024*1024))

	stream, _, err := client.GetStream(video, format)
	if err != nil {
		return err
	}
	defer stream.Close()

	// File သိမ်းဆည်းခြင်း - save to outputDir
	outputFile := filepath.Join(outputDir, baseName+".mp4")
	out, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer out.Close()

	// Progress tracking
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if totalSize > 0 {
				percent := float64(downloaded) / float64(totalSize) * 100
				fmt.Printf("\r⬇️  Downloading: %.1f%% (%.2f MB / %.2f MB)", percent, float64(downloaded)/(1024*1024), float64(totalSize)/(1024*1024))
			} else {
				fmt.Printf("\r⬇️  Downloaded: %.2f MB", float64(downloaded)/(1024*1024))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	fmt.Println() // New line after progress

	return nil
}

// Sanitize file name - remove special characters
func sanitizeFileName(name string) string {
	// Replace spaces and special chars with underscores
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		if r == ' ' {
			return '_'
		}
		return -1 // remove other characters
	}, name)
	// Limit length
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}

// getProjectDir returns the current working directory
func getProjectDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

// Speech-to-Text (Whisper အသုံးပြုခြင်း)
func speechToText(audioFile, outputFile string) (string, error) {
	// Whisper CLI သုံးခြင်း (Python Whisper ထည့်သွင်းရမည်)
	whisperPath := filepath.Join(filepath.Dir(os.Args[0]), "..", ".venv", "bin", "whisper")
	// If running with go run, use current working directory
	if _, err := os.Stat(whisperPath); os.IsNotExist(err) {
		whisperPath = filepath.Join(getProjectDir(), ".venv", "bin", "whisper")
	}

	// Get the output directory from the outputFile path
	outputDir := filepath.Dir(outputFile)
	cmd := exec.Command(whisperPath, audioFile, "--language", "en", "--output_format", "txt", "--output_dir", outputDir)

	// Pipe stdout and stderr to show progress in real-time
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("whisper error: %w", err)
	}

	// Output text file ဖတ်ခြင်း - whisper creates file based on input filename
	baseNameOnly := strings.TrimSuffix(filepath.Base(audioFile), filepath.Ext(audioFile))
	txtFile := filepath.Join(outputDir, baseNameOnly+".txt")
	text, err := os.ReadFile(txtFile)
	if err != nil {
		return "", err
	}

	// Save to output file (rename to _english.txt)
	if err := os.WriteFile(outputFile, text, 0644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", outputFile, err)
	}

	return string(text), nil
}

// Translation (Google Translate API သုံးခြင်း)
func translateToBurmese(text, outputFile string) (string, error) {
	// Google Translate API (အခမ်းအံ့ option)
	// ၎င်းအတွက် API key လိုအပ်ပါသည်

	// Alternative: Python subprocess သုံးခြင်း (deep-translator အသုံးပြု - အခမဲ့)
	pythonPath := filepath.Join(filepath.Dir(os.Args[0]), "..", ".venv", "bin", "python3")
	// If running with go run, use current working directory
	if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
		pythonPath = filepath.Join(getProjectDir(), ".venv", "bin", "python3")
	}

	// Split text into chunks of max 4500 characters (under 5000 limit)
	// Split on sentence boundaries where possible
	chunks := splitTextIntoChunks(text, 4500)
	var translatedChunks []string

	for i, chunk := range chunks {
		fmt.Printf("  Translating chunk %d/%d...\n", i+1, len(chunks))

		cmd := exec.Command(pythonPath, "-c", `
import sys
from deep_translator import GoogleTranslator
translator = GoogleTranslator(source='en', target='my')
text = sys.stdin.read()
result = translator.translate(text)
print(result)
`)

		// Stream stderr to show progress in real-time
		cmd.Stderr = os.Stderr

		// Set up stdin pipe to send text
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return "", fmt.Errorf("stdin pipe error: %w", err)
		}

		// Capture stdout for the result
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return "", fmt.Errorf("stdout pipe error: %w", err)
		}

		if err := cmd.Start(); err != nil {
			return "", fmt.Errorf("start error: %w", err)
		}

		// Write text to stdin and close
		_, err = stdin.Write([]byte(chunk))
		if err != nil {
			return "", fmt.Errorf("write error: %w", err)
		}
		stdin.Close()

		// Read the output
		output, err := io.ReadAll(stdout)
		if err != nil {
			return "", fmt.Errorf("read error: %w", err)
		}

		if err := cmd.Wait(); err != nil {
			// Fallback: အင်္ဂလိပ်စာ ပြန်ပေးခြင်း
			translatedChunks = append(translatedChunks, "Translation error - "+chunk)
			continue
		}

		translatedChunks = append(translatedChunks, strings.TrimSpace(string(output)))
	}

	result := strings.Join(translatedChunks, " ")

	// Save to output file
	if err := os.WriteFile(outputFile, []byte(result), 0644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", outputFile, err)
	}

	return result, nil
}

// splitTextIntoChunks splits text into chunks of maxSize characters
// trying to split on sentence boundaries
func splitTextIntoChunks(text string, maxSize int) []string {
	if len(text) <= maxSize {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= maxSize {
			chunks = append(chunks, remaining)
			break
		}

		// Find a good split point (end of sentence) within maxSize
		chunk := remaining[:maxSize]
		splitPoint := maxSize

		// Try to find sentence ending (.!?) followed by space
		for i := maxSize - 1; i > maxSize/2; i-- {
			if (chunk[i] == '.' || chunk[i] == '!' || chunk[i] == '?') &&
				(i+1 >= len(chunk) || chunk[i+1] == ' ' || chunk[i+1] == '\n') {
				splitPoint = i + 1
				break
			}
		}

		// If no sentence boundary found, try to split on space
		if splitPoint == maxSize {
			for i := maxSize - 1; i > maxSize/2; i-- {
				if chunk[i] == ' ' {
					splitPoint = i + 1
					break
				}
			}
		}

		chunks = append(chunks, strings.TrimSpace(remaining[:splitPoint]))
		remaining = strings.TrimSpace(remaining[splitPoint:])
	}

	return chunks
}

// Result များကို File သိမ်းဆည်းခြင်း
func saveResults(english, burmese, filename string) error {
	content := fmt.Sprintf("=== YouTube Speech-to-Text Results ===\n\nEnglish:\n%s\n\nBurmese:\n%s\n", english, burmese)
	return os.WriteFile(filename, []byte(content), 0644)
}

// getVoiceName returns the Edge TTS voice based on VOICE_PRESENTER env value
// Options: men/thiha -> male voice, women/girl -> female voice
// Default: men (male voice)
func getVoiceName() string {
	presenter := strings.ToLower(os.Getenv("VOICE_PRESENTER"))
	switch presenter {
	case "women", "girl":
		return "my-MM-NilarNeural" // အမျိုးသမီးအသံ
	case "men", "thiha", "":
		return "my-MM-ThihaNeural" // အမျိုးသားအသံ
	default:
		return "my-MM-ThihaNeural" // default: အမျိုးသားအသံ
	}
}

// Text-to-Speech for Burmese (Edge TTS အသုံးပြု - အရည်အသွေးပိုကောင်း)
func textToSpeechBurmese(textFile, outputAudio string) error {
	edgeTTSPath := filepath.Join(filepath.Dir(os.Args[0]), "..", ".venv", "bin", "edge-tts")
	if _, err := os.Stat(edgeTTSPath); os.IsNotExist(err) {
		edgeTTSPath = filepath.Join(getProjectDir(), ".venv", "bin", "edge-tts")
	}

	voiceName := getVoiceName()
	fmt.Printf("🔊 Generating Burmese audio with Edge TTS (voice: %s)...\n", voiceName)

	// Use Edge TTS with Myanmar voice
	// --rate: speech speed (-50% to +100%), --pitch: voice pitch
	cmd := exec.Command(edgeTTSPath,
		"--voice", voiceName,
		"--file", textFile,
		"--write-media", outputAudio,
		"--rate=-10%", // slightly slower for clarity
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("edge-tts error: %w", err)
	}

	fmt.Printf("✅ Audio saved to %s\n", outputAudio)
	return nil
}

// Merge Burmese audio with video (ffmpeg အသုံးပြု)
func mergeAudioWithVideo(videoFile, audioFile, outputFile string) error {
	fmt.Println("🎬 Merging Burmese audio with video...")

	// ffmpeg -i video.mp4 -i burmese_audio.mp3 -c:v copy -map 0:v:0 -map 1:a:0 output.mp4
	cmd := exec.Command("ffmpeg", "-y",
		"-i", videoFile,
		"-i", audioFile,
		"-c:v", "copy",
		"-map", "0:v:0",
		"-map", "1:a:0",
		"-shortest",
		outputFile,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %w", err)
	}

	fmt.Printf("✅ Video with Burmese audio saved to: %s\n", outputFile)
	return nil
}
