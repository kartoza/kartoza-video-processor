package cmd

import (
	"fmt"

	"github.com/kartoza/kartoza-video-processor/internal/recorder"
	"github.com/spf13/cobra"
)

var (
	monitorName   string
	noAudio       bool
	noWebcam      bool
	hwAccel       bool
	outputDir     string
	webcamDevice  string
	webcamFPS     int
	audioDevice   string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start screen recording",
	Long: `Start a new screen recording session.

The recording will capture:
  - Video from the specified monitor (or the one with the cursor)
  - Audio from the default input device (unless --no-audio is set)
  - Webcam video if available (unless --no-webcam is set)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rec := recorder.New()

		if rec.IsRecording() {
			return fmt.Errorf("recording is already in progress")
		}

		opts := recorder.Options{
			Monitor:      monitorName,
			NoAudio:      noAudio,
			NoWebcam:     noWebcam,
			HWAccel:      hwAccel,
			OutputDir:    outputDir,
			WebcamDevice: webcamDevice,
			WebcamFPS:    webcamFPS,
			AudioDevice:  audioDevice,
		}

		fmt.Println("Starting recording...")
		return rec.StartWithOptions(opts)
	},
}

func init() {
	startCmd.Flags().StringVarP(&monitorName, "monitor", "m", "", "Monitor name to record (default: monitor with cursor)")
	startCmd.Flags().BoolVar(&noAudio, "no-audio", false, "Disable audio recording")
	startCmd.Flags().BoolVar(&noWebcam, "no-webcam", false, "Disable webcam recording")
	startCmd.Flags().BoolVar(&hwAccel, "hw-accel", false, "Enable hardware acceleration (VAAPI)")
	startCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory (default: ~/Videos/Screencasts)")
	startCmd.Flags().StringVar(&webcamDevice, "webcam-device", "", "Webcam device (default: auto-detect)")
	startCmd.Flags().IntVar(&webcamFPS, "webcam-fps", 60, "Webcam framerate")
	startCmd.Flags().StringVar(&audioDevice, "audio-device", "@DEFAULT_SOURCE@", "PipeWire audio device")
}
