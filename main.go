package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/ticccccc/bdown/models"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("please input video list file!\nbdown videofile!")
		return
	}

	pth := os.Args[1]
	if _, err := os.Stat(pth); os.IsNotExist(err) {
		fmt.Println(pth + " dont exist!")
		return
	}

	vs := []string{}

	if ct, err := os.ReadFile(pth); err != nil {
		fmt.Println(pth + " read info fail")
		return
	} else {
		vs = strings.Split(string(ct), "\n")
	}

	if len(vs) == 0 {
		fmt.Println("no video in " + pth)
		return
	}

	if _, err := os.Stat("data"); os.IsNotExist(err) {
		os.MkdirAll("data", os.ModePerm)
	}

	sout := "data/success." + strings.TrimSuffix(path.Base(pth), path.Ext(pth)) + ".json"
	fout := "data/fail." + strings.TrimSuffix(path.Base(pth), path.Ext(pth)) + ".json"
	ffout := "data/fail." + strings.TrimSuffix(path.Base(pth), path.Ext(pth)) + ".csv"
	sfw, err := os.Create(sout)
	if err != nil {
		fmt.Println("create " + sout + " fail")
		return
	}
	defer sfw.Close()

	ffw, err := os.Create(fout)
	if err != nil {
		fmt.Println("create " + fout + " fail")
		return
	}
	defer ffw.Close()

	fffw, err := os.Create(ffout)
	if err != nil {
		fmt.Println("create " + ffout + " fail")
		return
	}
	defer fffw.Close()

	// 携程池
	pool := make(chan int, 100)
	// 处理完成交付
	pc := make(chan models.PlayInfo)
	// 完成信号
	done := make(chan int)

	sp := []models.PlayInfo{}
	fp := []models.PlayInfo{}

	go func() {
		// 分批处理
		for k := range vs {
			pool <- 1
			p := models.NewPlayInfo(vs[k])
			process(pool, pc, p)
		}
	}()

	go func(pc chan models.PlayInfo, done chan int, l int) {
		for i := 0; i < l; i++ {
			p := <-pc
			if p.Mp4 == "" {
				fp = append(fp, p)
			} else {
				sp = append(sp, p)
			}
		}
		done <- 1
	}(pc, done, len(vs))

	<-done
	ms, err := json.Marshal(sp)
	if err != nil {
		fmt.Println("json Marshal videos fail")
		return
	}
	sfw.WriteString(string(ms))

	mf, err := json.Marshal(fp)
	if err != nil {
		fmt.Println("json Marshal videos fail")
		return
	}
	ffw.WriteString(string(mf))

	for _, p := range fp {
		fffw.WriteString(p.Bvid + "\n")
	}
}

func process(pool chan int, pc chan models.PlayInfo, p models.PlayInfo) {
	defer func() { <-pool }()
	defer func() { pc <- p }()

	if err := p.GetInfo(); err != nil {
		fmt.Println(p.Bvid + " get cid fail")
		return
	}

	if err := p.GetPlay(); err != nil {
		fmt.Println(p.Bvid + " get mp4 url fail")
		return
	}

	if err := p.Download(); err != nil {
		fmt.Println(p.Bvid + " download mp4 fail")
		return
	}
}
