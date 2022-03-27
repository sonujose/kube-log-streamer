package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

var KubeClient *kubernetes.Clientset

func main() {

	r := gin.Default()
	r.LoadHTMLFiles("index.html")

	KubeClient, _ = NewKubeClient()

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	r.GET("/ws", func(c *gin.Context) {
		GetPodLogs(c.Writer, c.Request, "kong-dev", "kong-dev-kong-d8c494fcb-dmrtm", "proxy", true)
	})

	r.Run("localhost:12312")
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func GetPodLogs(w http.ResponseWriter, r *http.Request, namespace string, podName string, containerName string, follow bool) error {
	conn, err := wsupgrader.Upgrade(w, r, nil)

	if err != nil {
		fmt.Printf("Failed to set websocket upgrade: %v \n", err)
		return err
	}
	count := int64(100)
	podLogOptions := v1.PodLogOptions{
		Container: containerName,
		Follow:    follow,
		TailLines: &count,
	}

	podLogRequest := KubeClient.CoreV1().
		Pods(namespace).
		GetLogs(podName, &podLogOptions)

	stream, err := podLogRequest.Stream(context.TODO())
	if err != nil {
		return err
	}

	defer stream.Close()

	scanner := bufio.NewScanner(stream)

	for {

		for scanner.Scan() {
			conn.WriteJSON(scanner.Text())
			fmt.Println(scanner.Text()) // Println will add back the final '\n'
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("reading standard input: %v", err)
		}

		buf := make([]byte, 1000)
		numBytes, err := stream.Read(buf)

		if numBytes == 0 {
			continue
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		message := string(buf[:numBytes])
		fmt.Print(message)
		//conn.WriteJSON(message)
	}
	return nil
}
