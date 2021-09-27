package goline

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"

	goio "github.com/t-okkn/goio"
	_ "io/ioutil" // for debug
)

const (
	LINE_NOTIFY_URL string = "https://notify-api.line.me/api/notify"
)

type LineNotifyToken struct {
	Token  string
	Tag    string
}

var (
	notificationDisabled bool
)

func NewNotifyClientWithTag(token, tag string) *LineNotifyToken {
	notificationDisabled = false

	return &LineNotifyToken{token, tag}
}

func NewNotifyClient(token string) *LineNotifyToken {
	return NewNotifyClientWithTag(token, "")
}

func LineNotificationOn() {
	notificationDisabled = false
}

func LineNotificationOff() {
	notificationDisabled = true
}

func (t *LineNotifyToken) SendMessage(msg string) error {
	form := url.Values{}
	return t.sendFormData(msg, form)
}

func (t *LineNotifyToken) SendImageUrl(msg, imgurl, thumbUrl string) error {
	form := url.Values{}
	form.Add("imageFullsize", imgurl)
	form.Add("imageThumbnail", thumbUrl)

	return t.sendFormData(msg, form)
}

func (t *LineNotifyToken) SendSticker(msg string, packageId, id int32) error {
	if packageId < 1 || packageId > 4 {
		e := "スタンプのパッケージ識別子が不正です"
		return errors.New(e)
	}

	if !((1 <= id && id <= 47) || (100 <= id && id <= 307) ||
		(401 <= id && id <= 430) || (501 <= id && id <= 527) ||
		(601 <= id && id <= 632)) {

		e := "スタンプの識別子が不正です"
		return errors.New(e)
	}

	form := url.Values{}
	form.Add("stickerPackageId", fmt.Sprintf("%d", packageId))
	form.Add("stickerId", fmt.Sprintf("%d", id))

	return t.sendFormData(msg, form)
}

func (t *LineNotifyToken) SendImageFile(msg, imgpath string) error {
	if len([]rune(msg)) > 1000 {
		e := "1000文字より多いメッセージは送信できません"
		return errors.New(e)
	}

	if len(imgpath) <= 0 {
		e := "画像ファイルのPathを指定してください"
		return errors.New(e)
	}

	if exist, _ := goio.FileExists(imgpath); !exist {
		e := "画像ファイルが存在しません"
		return errors.New(e)
	}

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	if len(t.Tag) > 0 {
		msg = fmt.Sprintf("[%s]\n", t.Tag) + msg
	}

	if err := w.WriteField("message", msg); err != nil {
		return err
	}

	b := fmt.Sprintf("%t", notificationDisabled)
	if err := w.WriteField("notificationDisabled", b); err != nil {
		return err
	}

	_, filename := filepath.Split(imgpath)
	ext := filepath.Ext(imgpath)

	part := make(textproto.MIMEHeader)
	part.Set("Content-Disposition",
				`form-data; name="imageFile"; filename=` + filename)

	if ext == ".jpeg" || ext == ".jpg" ||
		ext == ".JPEG" || ext == ".JPG" {

		part.Set("Content-Type", "image/jpeg")

	} else if ext == ".png" || ext == ".PNG" {
		part.Set("Content-Type", "image/png")

	} else {
		e := "LINE Notifyでは[jpeg/png]形式の画像のみをサポートしています"
		return errors.New(e)
	}

	stream, err := os.Open(imgpath)
	if err != nil {
		return err
	}
	defer stream.Close()

	// バイナリから画像判定をするとなぜかLINEから400でレスポンスが返る
	//img := bytes.NewBuffer(nil)
	//tr := io.TeeReader(stream, img)
	//_, format, err := image.DecodeConfig(tr)
	//if err != nil { return err }

	if fw, err := w.CreatePart(part); err != nil {
		return err

	} else {
		if _, err := io.Copy(fw, stream); err != nil {
			return err
		}
	}

	// boundaryの書き込み
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", LINE_NOTIFY_URL, body)
	if err != nil { return err }

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer " + t.Token)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil { return err }

	if resp.StatusCode != 200 {
		e := "[status code: " + resp.Status + "] Failed to send data"
		return errors.New(e)
	}

	/* ***********for debug*********** //
	defer resp.Body.Close()
	fmt.Println(resp.Header)
	fmt.Println("*******")
	str, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(str))
	// ******************************* */

	return nil
}

func (t *LineNotifyToken) sendFormData(msg string, form url.Values) error {
	if len([]rune(msg)) > 1000 {
		e := errors.New("1000文字より多いメッセージは送信できません")
		return e
	}

	if len(t.Tag) > 0 {
		msg = fmt.Sprintf("[%s]\n", t.Tag) + msg
	}

	form.Add("message", msg)

	strbool := fmt.Sprintf("%t", notificationDisabled)
	form.Add("notificationDisabled", strbool)

	requri, err := url.ParseRequestURI(LINE_NOTIFY_URL)
	if err != nil {
		return err
	}

	body := strings.NewReader(form.Encode())
	req, err := http.NewRequest("POST", requri.String(), body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer " + t.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		e := "[status code: " + resp.Status + "] Failed to send data"
		return errors.New(e)
	}

	/* ***********for debug*********** //
	defer resp.Body.Close()
	fmt.Println(resp.Header)
	fmt.Println("*******")
	str, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(str))
	// ******************************* */

	return nil
}

