package main

import "video-handler-microservice/internal/converter"

func main() {
	/*directory := "cmd/mediatest/media/upload/1"
	dirPath := "merged-1.mp4"

	dir, _ := os.Getwd()

	fmt.Printf("Project directory %v\n", dir)

	err := mergeChunks(directory, dirPath)

	if err != nil {
		fmt.Println(err)
		return
	}*/

	vc := converter.NewVideoConverter()
	vc.Handle([]byte(`{"video_id": 1, "path": "/media/uploads/1"}`))
}
