package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

func getY2MateID(vid string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	fotmatReq := strings.NewReader(fmt.Sprintf("url=https://www.youtube.com/watch?v=%s&q_auto=0&ajax=1", vid))
	req, err := http.NewRequestWithContext(ctx, "POST", "https://www.y2mate.com/mates/en249/analyze/ajax", fotmatReq)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	jsonRet := map[string]interface{}{}
	defer resp.Body.Close()
	_body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(_body, jsonRet); err != nil {
		return "", err
	}
	if val, ok := jsonRet["result"]; ok {
		re := regexp.MustCompile(`k__id\s+=\s+(["'])(.*?)\1`)
		ret := re.Find([]byte(val.(string)))
		if ret == nil {
			return "", fmt.Errorf("Cannot Parse URL")
		}
		_ret := strings.ReplaceAll(string(ret), `k__id = "`, "")
		_ret = strings.ReplaceAll(_ret, `"`, "")
		return _ret, nil
	}
	return "", fmt.Errorf("Cannot get y2mate id!")
}

func getConvert(vid, y2id string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	fotmatReq := strings.NewReader(fmt.Sprintf("type=youtube&_id=%s&v_id=%s&ajax=1&token=&ftype=mp3&fquality=128", y2id, vid))
	req, err := http.NewRequestWithContext(ctx, "POST", "https://www.y2mate.com/mates/convert", fotmatReq)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	jsonRet := map[string]interface{}{}
	defer resp.Body.Close()
	_body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(_body, jsonRet); err != nil {
		return "", err
	}
	if val, ok := jsonRet["result"]; ok {
		re := regexp.MustCompile(`<a\s+(?:[^>]*?\s+)?href=(["'])(.*?)\1`)
		ret := re.Find([]byte(val.(string)))
		if ret == nil {
			return "", fmt.Errorf("Cannot Parse URL")
		}
		_ret := strings.ReplaceAll(string(ret), `<a href=`, "")
		_ret = strings.ReplaceAll(_ret, `"`, "")
		return _ret, nil
	}
	return "", fmt.Errorf("Cannot get video")
}

func GetYoutube(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	vid := ps.ByName("vid")
	id, err := getY2MateID(vid)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		log.Printf("%v", err)
		return
	}
	link, err := getConvert(vid, id)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		log.Printf("%v", err)
		return
	}
	pr, pw := io.Pipe()
	//Async Fetch the audio
	go func() {
		_ctx, _cancel := context.WithTimeout(context.Background(), 1*time.Hour)
		defer _cancel()
		client := &http.Client{}

		req, err := http.NewRequestWithContext(_ctx, "GET", link, nil)
		if err != nil {
			log.Fatalln(err)
			return
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36")

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalln(err)
			return
		}

		defer resp.Body.Close()
		io.Copy(pw, resp.Body)
	}()
	io.Copy(w, pr)
}

func shutdown(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server Shutdown: %v", err)
	}
}

func New(ctx context.Context, httpaddr string) {
	router := httprouter.New()
	router.GET("/ytb/:vid", GetYoutube)

	shutdownSignal := make(chan struct{})
	srv := &http.Server{
		Handler: router,
		Addr:    httpaddr,
	}
	go func() {
		defer close(shutdownSignal)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			return
		}
	}()
	<-ctx.Done()
	shutdown(srv)
	<-shutdownSignal
}
