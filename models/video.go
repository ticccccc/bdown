package models

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type PlayInfo struct {
	Bvid     string `json:"bvid"` // bvid
	Cid      string `json:"cid"`
	Duration int32  `json:"duration"`
	Width    int    `json:"width"`
	Hight    int    `json:"height"`
	Rotate   int    `json:"rotate"`
	Size     int    `json:"size"`
	Url      string `json:"url"`
	Mp4      string `json:"mp4"`
}

func NewPlayInfo(bvid string) PlayInfo {
	return PlayInfo{
		Bvid: bvid,
	}
}

// bvid 转 cid (视频的content id)
func (p *PlayInfo) GetInfo() error {
	params := url.Values{}
	params.Set("bvid", p.Bvid)
	u := "https://api.bilibili.com/x/player/pagelist?" + params.Encode()

	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.13; rv:56.0) Gecko/20100101 Firefox/56.0")
	req.Header.Set("Host", "api.bilibili.com")
	req.Header.Set("Referer", "https://www.bilibili.com/video/"+p.Bvid)
	req.Header.Set("Origin", "https://www.bilibili.com")

	client := http.Client{
		Timeout: time.Duration(2 * time.Second),
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("get playinfo of " + p.Bvid + " fail: " + err.Error())
		return errors.New("request execute fail: " + err.Error())
	}

	if resp.StatusCode != 200 {
		fmt.Println("get playinfo of " + p.Bvid + " fail: get statuscode(" + strconv.Itoa(resp.StatusCode) + ")")
		return errors.New("server status abnormal: get statuscode(" + strconv.Itoa(resp.StatusCode) + ")")
	}
	defer resp.Body.Close()

	type CInfo struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
		Data []struct {
			Cid       int64 `json:"cid"`
			Duration  int32 `json:"duration"`
			Dimension struct {
				Width  int `json:"width"`
				Hight  int `json:"height"`
				Rotate int `json:"rotate"`
			} `json:"dimension"`
		} `json:"data"`
	}

	var c CInfo
	if err = json.NewDecoder(resp.Body).Decode(&c); err != nil {
		fmt.Println("parse playinfo error info: " + err.Error() + " bvid: " + p.Bvid)
		return errors.New("parse json fail: " + err.Error())
	}

	if len(c.Data) != 1 {
		j, _ := json.Marshal(c.Data)
		fmt.Println("playinfo has more than 1 cid: " + p.Bvid + " " + string(j))
		return errors.New("playinfo has more than 1 cid: " + p.Bvid)
	}

	p.Cid = strconv.FormatInt(c.Data[0].Cid, 10)
	p.Duration = c.Data[0].Duration
	p.Hight = c.Data[0].Dimension.Hight
	p.Width = c.Data[0].Dimension.Width
	p.Rotate = c.Data[0].Dimension.Rotate

	return nil
}

// 使用cid获取播放地址
func (p *PlayInfo) GetPlay() error {
	// url加密
	ak := string([]byte{105, 86, 71, 85, 84, 106, 115, 120, 118, 112, 76, 101, 117, 68, 67, 102})
	sk := string([]byte{97, 72, 82, 109, 104, 87, 77, 76, 107, 100, 101, 77, 117, 73, 76, 113, 79, 82, 110, 89, 90, 111, 99, 119, 77, 66, 112, 77, 69, 79, 100, 116})
	params := url.Values{}
	params.Set("appkey", ak)
	params.Set("cid", p.Cid)
	params.Set("otype", "json")
	// 1080p+:112;1080p:80;720p:64;480p:32;360p:16)
	params.Set("quality", "80")
	b := md5.Sum([]byte(params.Encode() + sk))

	chksum := hex.EncodeToString(b[:])
	params.Set("sign", chksum)
	u := "https://interface.bilibili.com/v2/playurl?" + params.Encode()

	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.13; rv:56.0) Gecko/20100101 Firefox/56.0")
	req.Header.Set("host", "api.bilibili.com")
	req.Header.Set("refer", "https://api.bilibili.com/x/web-interface/view?bvid="+p.Bvid)
	req.Header.Set("origin", "https://www.bilibili.com")

	client := http.Client{
		Timeout: time.Duration(2 * time.Second),
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("get playinfo of " + p.Bvid + " fail: " + err.Error())
		return errors.New("request execute fail: " + err.Error())
	}

	if resp.StatusCode != 200 {
		fmt.Println("get playinfo of " + p.Bvid + " fail: get statuscode(" + strconv.Itoa(resp.StatusCode) + ")")
		return errors.New("server status abnormal: get statuscode(" + strconv.Itoa(resp.StatusCode) + ")")
	}
	defer resp.Body.Close()

	type CInfo struct {
		Durl []struct {
			Url    string `json:"url"`
			Size   int32  `json:"size"`
			Length int32  `json:"length"`
		} `json:"durl"`
	}
	var cinfo CInfo

	err = json.NewDecoder(resp.Body).Decode(&cinfo)
	if err != nil {
		fmt.Println(p.Bvid + " parse video playurl fail")
		return err
	}

	if len(cinfo.Durl) < 1 {
		fmt.Println(p.Bvid + " download url parse fail")
		return errors.New(p.Bvid + " download url parse fail")
	}
	p.Url = cinfo.Durl[0].Url
	p.Size = int(cinfo.Durl[0].Size)
	return nil
}

// 下载视频
func (p *PlayInfo) Download() error {
	req, _ := http.NewRequest("GET", p.Url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.13; rv:56.0) Gecko/20100101 Firefox/56.0")
	// req.Header.Set("host", "upos-hz-mirrorks3.acgvideo.com")
	req.Header.Set("Referer", "https://api.bilibili.com/x/web-interface/view?bvid="+p.Bvid)
	req.Header.Set("Origin", "https://www.bilibili.com")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Range", "bytes=0-")
	req.Header.Set("Connection", "keep-alive")

	client := http.Client{
		Timeout: time.Duration(10 * 60 * time.Second),
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("get download of " + p.Bvid + " fail: " + err.Error())
		return errors.New("request execute fail: " + err.Error())
	}

	if resp.StatusCode > 299 {
		fmt.Println("get download of " + p.Bvid + " fail: get statuscode(" + strconv.Itoa(resp.StatusCode) + ")")
		return errors.New("server status abnormal: get statuscode(" + strconv.Itoa(resp.StatusCode) + ")")
	}
	defer resp.Body.Close()

	f, err := os.Create("data/" + p.Bvid + ".flv")
	if err != nil {
		fmt.Println("create file fail")
		return err
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		fmt.Println("download mp4 failed: " + p.Bvid)
		return err
	}
	p.Mp4 = "data/" + p.Bvid + ".flv"
	return nil
}
