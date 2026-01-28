package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
)

var liveToBurmeseCmd = &cobra.Command{
	Use:   "live",
	Short: "Live English audio to Burmese translation",
	Long:  "Translate live English speech to Burmese audio in real-time.",
	Run: func(cmd *cobra.Command, args []string) {
		live()
	},
}

func init() {
	rootCmd.AddCommand(liveToBurmeseCmd)
}

const (
	liveSampleRate = 16000
	liveChannels   = 1
	chunkDuration  = 5 // seconds per chunk for real-time processing
)

func live() {
	fmt.Println("🎤 တိုက်ရိုက် ဘာသာပြန်စနစ် စတင်နေသည်...")
	fmt.Println("📢 English စကားပြောပါ - မြန်မာလို ပြန်ပေးပါမည်")
	fmt.Println("⏹️  ရပ်ရန် Ctrl+C နှိပ်ပါ")
	fmt.Println(strings.Repeat("─", 50))

	projectDir := "/home/banyar-sithu/FrontiirProjects/video"

	// Handle Ctrl+C for graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	running := true
	var wg sync.WaitGroup

	// Continuous recording and translation loop
	chunkNum := 0
	for running {
		select {
		case <-stopChan:
			fmt.Println("\n\n⏹️ ရပ်တန့်သည်...")
			running = false
			continue
		default:
		}

		chunkNum++
		audioFile := filepath.Join(projectDir, fmt.Sprintf("chunk_%d.wav", chunkNum))

		// Record audio chunk
		fmt.Printf("\n🔴 [%d] Recording...\n", chunkNum)
		err := recordChunk(audioFile, chunkDuration)
		if err != nil {
			fmt.Printf("❌ Recording error: %v\n", err)
			continue
		}

		// Process in background while recording next chunk
		wg.Add(1)
		go func(file string, num int) {
			defer wg.Done()
			defer os.Remove(file)

			processChunk(file, num)
		}(audioFile, chunkNum)
	}

	// Wait for all processing to complete
	wg.Wait()
	fmt.Println("\n✅ ပြီးစီးပါပြီ")
}

// Record a single audio chunk
func recordChunk(outputFile string, duration int) error {
	os.Remove(outputFile)

	cmd := exec.Command("arecord",
		"-f", "S16_LE",
		"-r", fmt.Sprintf("%d", liveSampleRate),
		"-c", fmt.Sprintf("%d", liveChannels),
		"-t", "wav",
		"-d", fmt.Sprintf("%d", duration),
		"-q", // quiet mode
		outputFile,
	)

	return cmd.Run()
}

// Process a single chunk: transcribe, translate, speak
func processChunk(audioFile string, chunkNum int) {
	// Speech-to-Text
	englishText, err := liveConvertSpeechToEnglish(audioFile)
	if err != nil {
		return
	}

	if strings.TrimSpace(englishText) == "" {
		return
	}

	fmt.Printf("🗣️ [%d] EN: %s\n", chunkNum, englishText)

	// Translate to Burmese
	burmeseText, err := liveTranslateToBurmese(englishText)
	if err != nil {
		fmt.Printf("❌ Translation error: %v\n", err)
		return
	}

	fmt.Printf("🔤 [%d] MY: %s\n", chunkNum, burmeseText)

	// Text-to-Speech
	liveSpeakBurmese(burmeseText)
}

// Process live translation
func processLiveTranslation(audioFile string) {
	// Speech-to-Text using Whisper
	englishText, err := liveConvertSpeechToEnglish(audioFile)
	if err != nil {
		fmt.Println("❌ Whisper Error:", err)
		return
	}

	if strings.TrimSpace(englishText) == "" {
		fmt.Println("⚠️ No speech detected")
		return
	}

	fmt.Printf("🗣️ English: %s\n", englishText)

	// Translate to Burmese
	burmeseText, err := liveTranslateToBurmese(englishText)
	if err != nil {
		fmt.Println("❌ Translation Error:", err)
		return
	}

	fmt.Printf("🔤 Burmese: %s\n", burmeseText)

	// Text-to-Speech
	fmt.Println("\n🔊 Burmese အသံထွက်နေသည်...")
	err = liveSpeakBurmese(burmeseText)
	if err != nil {
		fmt.Println("❌ TTS Error:", err)
		return
	}

	fmt.Println("✅ ပြီးစီးပါပြီ")
}

// Speech-to-Text using Whisper
func liveConvertSpeechToEnglish(audioFile string) (string, error) {
	whisperPath := "/home/banyar-sithu/FrontiirProjects/video/.venv/bin/whisper"
	outputDir := filepath.Dir(audioFile)

	cmd := exec.Command(whisperPath, audioFile,
		"--language", "en",
		"--output_format", "txt",
		"--output_dir", outputDir)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("whisper failed: %w", err)
	}

	// Read output text file
	baseName := strings.TrimSuffix(filepath.Base(audioFile), ".wav")
	txtFile := filepath.Join(outputDir, baseName+".txt")
	text, err := os.ReadFile(txtFile)
	if err != nil {
		return "", err
	}

	os.Remove(txtFile) // Cleanup

	return strings.TrimSpace(string(text)), nil
}

// Translate to Burmese using deep_translator
func liveTranslateToBurmese(englishText string) (string, error) {
	pythonPath := filepath.Join(filepath.Dir(os.Args[0]), "..", ".venv", "bin", "python3")
	if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
		pythonPath = "/home/banyar-sithu/FrontiirProjects/video/.venv/bin/python3"
	}

	cmd := exec.Command(pythonPath, "-c", `
import sys
from deep_translator import GoogleTranslator
translator = GoogleTranslator(source='en', target='my')
text = sys.stdin.read()
result = translator.translate(text)
print(result)
`)

	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("stdin pipe error: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe error: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start error: %w", err)
	}

	_, err = stdin.Write([]byte(englishText))
	if err != nil {
		return "", fmt.Errorf("write error: %w", err)
	}
	stdin.Close()

	output, err := io.ReadAll(stdout)
	if err != nil {
		return "", fmt.Errorf("read error: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("translation failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// Text-to-Speech using Edge TTS
func liveSpeakBurmese(burmeseText string) error {
	edgeTTSPath := filepath.Join(filepath.Dir(os.Args[0]), "..", ".venv", "bin", "edge-tts")
	if _, err := os.Stat(edgeTTSPath); os.IsNotExist(err) {
		edgeTTSPath = "/home/banyar-sithu/FrontiirProjects/video/.venv/bin/edge-tts"
	}

	outputAudio := "live_output.mp3"

	// Generate audio using Edge TTS
	cmd := exec.Command(edgeTTSPath,
		"--voice", "my-MM-ThihaNeural",
		"--text", burmeseText,
		"--write-media", outputAudio,
		"--rate=-10%",
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("edge-tts error: %w", err)
	}

	// Play the audio
	playCmd := exec.Command("ffplay", "-nodisp", "-autoexit", outputAudio)
	playCmd.Run()

	os.Remove(outputAudio)

	return nil
}
