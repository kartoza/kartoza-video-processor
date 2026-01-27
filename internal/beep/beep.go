package beep

import (
	"fmt"
	"os/exec"
	"time"
)

// Descending frequencies for countdown beeps (Hz)
// 5=880Hz, 4=784Hz, 3=698Hz, 2=622Hz, 1=554Hz (descending A5 to C#5)
var Frequencies = map[int]int{
	5: 880,
	4: 784,
	3: 698,
	2: 622,
	1: 554,
}

// Play plays a beep at the specified frequency for the countdown number
func Play(count int) {
	freq, ok := Frequencies[count]
	if !ok {
		return
	}

	// Try different methods to play a beep

	// Method 1: Use pw-play with a generated tone (PipeWire)
	// Generate a short sine wave using ffmpeg and play it
	if tryFFmpegBeep(freq) {
		return
	}

	// Method 2: Use speaker-test (ALSA)
	if trySpeakerTest(freq) {
		return
	}

	// Method 3: Use paplay with a system sound (PulseAudio)
	if tryPaplay() {
		return
	}

	// Method 4: Console beep (may not work on all systems)
	fmt.Print("\a")
}

// tryFFmpegBeep generates and plays a tone using ffmpeg and pw-play/aplay
func tryFFmpegBeep(freq int) bool {
	// Generate a 100ms sine wave tone and pipe to audio player
	// Using pw-cat (PipeWire) or aplay (ALSA)
	duration := "0.1"

	// Try pw-cat first (PipeWire)
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf("ffmpeg -f lavfi -i 'sine=frequency=%d:duration=%s' -f wav - 2>/dev/null | pw-cat --playback - 2>/dev/null",
			freq, duration))
	if err := cmd.Run(); err == nil {
		return true
	}

	// Try aplay (ALSA)
	cmd = exec.Command("bash", "-c",
		fmt.Sprintf("ffmpeg -f lavfi -i 'sine=frequency=%d:duration=%s' -f wav - 2>/dev/null | aplay -q - 2>/dev/null",
			freq, duration))
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

// trySpeakerTest uses speaker-test to generate a tone
func trySpeakerTest(freq int) bool {
	cmd := exec.Command("speaker-test", "-t", "sine", "-f", fmt.Sprintf("%d", freq), "-l", "1", "-p", "1")
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Start()
	if err != nil {
		return false
	}

	// Kill after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = cmd.Process.Kill()
	}()

	return true
}

// tryPaplay plays a system sound using paplay
func tryPaplay() bool {
	// Try common system sound locations
	sounds := []string{
		"/usr/share/sounds/freedesktop/stereo/message.oga",
		"/usr/share/sounds/freedesktop/stereo/bell.oga",
		"/usr/share/sounds/sound-icons/bell.wav",
	}

	for _, sound := range sounds {
		cmd := exec.Command("paplay", sound)
		if err := cmd.Run(); err == nil {
			return true
		}
	}

	return false
}
