package main

import (
	"flag"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"ubvremux/demux"
	"ubvremux/ffmpegutil"
	"ubvremux/ubv"
)

var ReleaseVersion string
var GitCommit string

func main() {
	includeAudioPtr := flag.Bool("with-audio", false, "If true, extract audio")
	includeVideoPtr := flag.Bool("with-video", true, "If true, extract video")
	outputFolder := flag.String("output-folder", "./", "The path to output remuxed files to. \"SRC-FOLDER\" to put alongside .ubv files")
	remuxPtr := flag.Bool("mp4", true, "If true, will create an MP4 as output")
	versionPtr := flag.Bool("version", false, "Display version and quit")

	flag.Parse()

	// Perform some argument combo validation
	if *versionPtr {
		println("UBV Remux Tool")
		println("Copyright (c) Peter Wright 2020")
		println("https://github.com/petergeneric/unifi-protect-remux")
		println("")

		// If there's a release version specified, use that
		if len(ReleaseVersion) > 0 {
			println("\tVersion:    ", ReleaseVersion)
		}
		println("\tGit commit: ", GitCommit)

		os.Exit(0)
	} else if len(flag.Args()) == 0 {
		// Terminate immediately if no .ubv files were provided
		println("Expected at least one .ubv file as input!\n")

		flag.Usage()
		os.Exit(1)
	} else if !*includeAudioPtr && !*includeVideoPtr {
		// Fail if extracting neither audio nor video
		println("Must enable extraction of at least one of: audio, video!\n")

		flag.Usage()
		os.Exit(1)
	}

	for _, ubvFile := range flag.Args() {
		info := ubv.Analyse(ubvFile, *includeAudioPtr)

		log.Printf("\n\n*** Parsing complete! ***\n\n")
		log.Printf("Number of partitions: %d", len(info.Partitions))

		if len(info.Partitions) > 0 {
			log.Printf("First Partition:")
			log.Printf("\tTracks: %d", len(info.Partitions[0].Tracks))
			log.Printf("\tFrames: %d", len(info.Partitions[0].Frames))
			log.Printf("\tStart Timecode: %s", info.Partitions[0].Tracks[7].StartTimecode.Format(time.RFC3339))
		}

		for _, partition := range info.Partitions {
			var videoFile string
			var audioFile string
			var mp4 string
			{
				outputFolder := strings.TrimSuffix(*outputFolder, "/")

				if outputFolder == "SRC-FOLDER" {
					outputFolder = path.Dir(info.Filename)
				}

				// TODO generate a base filename that contains the start timecode?
				basename := outputFolder + "/" + strings.TrimSuffix(path.Base(ubvFile), path.Ext(ubvFile))

				// For multi-partition files, generate a file per partition
				// For single-partition files, we just use the simple output filename
				if len(info.Partitions) > 1 {
					basename = basename + "_p" + strconv.Itoa(partition.Index)
				}

				if *includeVideoPtr && partition.VideoTrackCount > 0 {
					videoFile = basename + ".h264"
				}

				if *includeAudioPtr && partition.AudioTrackCount > 0 {
					audioFile = basename + ".aac"
				}

				if *remuxPtr {
					mp4 = basename + ".mp4"
				}
			}

			demux.DemuxSinglePartitionToNewFiles(ubvFile, videoFile, audioFile, partition)

			if *remuxPtr {
				log.Println("Generating MP4 ", mp4, " from ", videoFile, " and ", audioFile)

				// Spawn FFmpeg to remux
				// TODO: if we do a little parsing of the bitstream we could create an MP4 and reduce write amplification
				ffmpegutil.MuxAudioAndVideo(partition, videoFile, audioFile, mp4)

				// Delete
				if len(videoFile) > 0 {
					if err := os.Remove(videoFile); err != nil {
						log.Println("Warning: could not delete ", videoFile+": ", err)
					}
				}
				if len(audioFile) > 0 {
					if err := os.Remove(audioFile); err != nil {
						log.Println("Warning: could not delete ", audioFile+": ", err)
					}
				}
			}
		}
	}
}
