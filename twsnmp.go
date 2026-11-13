package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"go.etcd.io/bbolt"
)

type Site struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Url      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"password"`
	Loc      string `json:"loc"`
	State    string `json:"state"`
	Type     string `json:"type"` // "" or "auto": 自動判別, "fc": TWSNMP FC, "neo": TWSNMP NEO
}

type Api struct {
	Id    string
	Url   string
	Token string
	State string
	Type  string
}

type Twsnmp struct {
	db      *bbolt.DB
	Sites   sync.Map
	api     sync.Map
	updater *updateManager
}

// GetVersion は現在のアプリバージョン情報を返します
func (t *Twsnmp) GetVersion() VersionInfo {
	if t.updater == nil {
		return resolveVersionInfo(version, commit)
	}
	return t.updater.version
}

// GetUpdateState は現在のアップデート状態を返します
func (t *Twsnmp) GetUpdateState() UpdateState {
	if t.updater == nil {
		return UpdateState{
			State:          "unconfigured",
			CurrentVersion: resolveVersionInfo(version, commit).Display,
		}
	}
	return t.updater.State()
}

// CheckUpdate は最新バージョンの有無を確認します
func (t *Twsnmp) CheckUpdate() UpdateState {
	if t.updater == nil {
		return t.GetUpdateState()
	}
	return t.updater.CheckUpdate()
}

// ApplyUpdate は更新をダウンロード・検証・適用し、アプリを再起動します
func (t *Twsnmp) ApplyUpdate() error {
	if t.updater == nil {
		return fmt.Errorf("updater not initialized")
	}
	return t.updater.ApplyUpdate()
}

func (t *Twsnmp) close() {
	if t.db != nil {
		_ = t.db.Close()
		t.db = nil
	}
}

func (t *Twsnmp) GetSites() []Site {
	ret := []Site{}
	t.Sites.Range(func(k, v any) bool {
		if s, ok := v.(*Site); ok {
			if av, ok := t.api.Load(s.Id); ok {
				if a, ok := av.(*Api); ok {
					s.State = a.State
					if s.Type == "" || s.Type == "auto" {
						if a.Type != "" && a.Type != "auto" {
							s.Type = a.Type
						}
					}
				}
			}
			ret = append(ret, *s)
		}
		return true
	})
	return ret
}

// UpdateSite は Siteを追加・更新します
func (t *Twsnmp) UpdateSite(id, name, url, user, password, siteType string) {
	state := "unknown"
	if id == "" {
		id = fmt.Sprintf("%x", time.Now().UnixNano())
	}
	if v, ok := t.Sites.Load(id); ok {
		if s, ok := v.(*Site); ok {
			state = s.State
		}
	}
	s := &Site{
		Id:       id,
		Name:     name,
		Url:      url,
		User:     user,
		Password: password,
		State:    state,
		Type:     siteType,
	}
	t.Sites.Store(id, s)
	t.api.Delete(id)
	t.api.Store(id, &Api{
		Id:   id,
		Url:  s.Url,
		Type: s.Type,
	})
	t.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("twsnmp"))
		if j, err := json.Marshal(s); err == nil {
			b.Put([]byte(id), j)
		} else {
			return err
		}
		return nil
	})
}

// DeleteSite は Siteを削除します
func (t *Twsnmp) DeleteSite(id string) {
	t.Sites.Delete(id)
	t.api.Delete(id)
	t.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("twsnmp"))
		return b.Delete([]byte(id))
	})
}

func (t *Twsnmp) UpdateSiteLoc(id, loc string) {
	v, ok := t.Sites.Load(id)
	if !ok {
		return
	}
	s, ok := v.(*Site)
	if !ok {
		return
	}
	s.Loc = loc
	t.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("twsnmp"))
		if j, err := json.Marshal(s); err == nil {
			b.Put([]byte(id), j)
		} else {
			return err
		}
		return nil
	})
}

func (t *Twsnmp) OpenSiteMap(id string) bool {
	v, ok := t.Sites.Load(id)
	if !ok {
		return false
	}
	s, ok := v.(*Site)
	if !ok {
		return false
	}

	title := "TWSNMP Map: " + s.Name
	resolvedType := s.Type
	if av, ok := t.api.Load(s.Id); ok {
		if a, ok := av.(*Api); ok && a.Type != "" && a.Type != "auto" {
			resolvedType = a.Type
		}
	}
	if resolvedType == "neo" {
		title = "TWSNMP NEO Map: " + s.Name
	} else if resolvedType == "fc" {
		title = "TWSNMP FC Map: " + s.Name
	}

	u, err := url.Parse(s.Url)
	if err != nil {
		return false
	}
	q := u.Query()
	// TWSNMP FCパラメータ
	q.Set("UserID", s.User)
	q.Set("Password", s.Password)
	// TWSNMP NEOパラメータ
	q.Set("user", s.User)
	q.Set("password", s.Password)
	if av, ok := t.api.Load(s.Id); ok {
		if a, ok := av.(*Api); ok && a.Token != "" {
			q.Set("token", a.Token)
		}
	}
	u.RawQuery = q.Encode()

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  title,
		Width:  1280,
		Height: 800,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
		},
		URL: u.String(),
	})
	return true
}

func (t *Twsnmp) LoadViewport() string {
	vp := ""
	t.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("config"))
		if b != nil {
			if v := b.Get([]byte("viewport")); v != nil {
				vp = string(v)
			}
		}
		return nil
	})
	return vp
}

func (t *Twsnmp) SaveViewport(vp string) {
	t.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("config"))
		if err != nil {
			log.Println(err)
			return err
		}
		return b.Put([]byte("viewport"), []byte(vp))
	})
}

func (t *Twsnmp) IsDark() bool {
	r := false
	t.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("config"))
		if b != nil {
			if v := b.Get([]byte("dark")); v != nil {
				r = string(v) == "true"
			}
		}
		return nil
	})
	return r
}

func (t *Twsnmp) SaveDark(r bool) {
	t.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("config"))
		if err != nil {
			log.Println(err)
			return err
		}
		v := "false"
		if r {
			v = "true"
		}
		return b.Put([]byte("dark"), []byte(v))
	})
}

func (t *Twsnmp) GetMapStyle() string {
	r := ""
	t.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("config"))
		if b != nil {
			if v := b.Get([]byte("mapStyle")); v != nil {
				r = string(v)
			}
		}
		return nil
	})
	return r
}

func (t *Twsnmp) SaveMapStyle(ms string) {
	t.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("config"))
		if err != nil {
			log.Println(err)
			return err
		}
		return b.Put([]byte("mapStyle"), []byte(ms))
	})
}

// loadはsiteの設定を読み込む
func (t *Twsnmp) load() {
	conf := os.Getenv("TWSNMPMW_CONFIG_DIR")
	if conf == "" {
		var err error
		conf, err = os.UserConfigDir()
		if err != nil {
			log.Fatal(err)
		}
	}
	os.MkdirAll(filepath.Join(conf, "twsnmpmw"), 0766)
	var err error
	t.db, err = bbolt.Open(filepath.Join(conf, "twsnmpmw", "twsnmpmw.db"), 0600, nil)
	if err != nil {
		log.Fatalf("Open err=%v", err)
	}
	t.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("twsnmp"))
		if b == nil {
			b, err = tx.CreateBucketIfNotExists([]byte("twsnmp"))
			if err != nil {
				log.Fatal(err)
			}
		}
		b.ForEach(func(k, v []byte) error {
			id := string(k)
			var s Site
			if err := json.Unmarshal(v, &s); err == nil && id == s.Id {
				t.Sites.Store(id, &s)
				t.api.Store(id, &Api{
					Id:   id,
					Url:  s.Url,
					Type: s.Type,
				})
			}
			return nil
		})
		return nil
	})
}

func (t *Twsnmp) checkSiteState() chan bool {
	ticker := time.NewTicker(10 * time.Second)
	stop := make(chan bool)
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				// Check site here
				log.Println("check site")
				t.Sites.Range(func(k, v any) bool {
					if s, ok := v.(*Site); ok {
						av, ok := t.api.Load(s.Id)
						if !ok {
							return true
						}
						a, ok := av.(*Api)
						if !ok {
							return true
						}
						if a.Token == "" {
							if !a.login(s.User, s.Password) {
								a.State = "unknown"
								return true
							}
						}
						a.GetState()
					}
					app.Event.Emit("update")
					return true
				})
			}
		}
	}()
	return stop
}

var insecureTransport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

var insecureClient = &http.Client{Transport: insecureTransport}

type fcLoginParam struct {
	UserID   string `json:"UserID"`
	Password string `json:"Password"`
}

type neoLoginParam struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type loginResp struct {
	Token string `json:"token"`
}

func (a *Api) login(user, password string) bool {
	switch a.Type {
	case "neo":
		return a.loginNeo(user, password)
	case "fc":
		return a.loginFC(user, password)
	default:
		// 自動判別: まず NEO を試行、失敗時に FC を試行
		if a.loginNeo(user, password) {
			a.Type = "neo"
			return true
		}
		if a.loginFC(user, password) {
			a.Type = "fc"
			return true
		}
		return false
	}
}

func (a *Api) loginNeo(user, password string) bool {
	p := neoLoginParam{User: user, Password: password}
	j, err := json.Marshal(&p)
	if err != nil {
		log.Println(err)
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequest(http.MethodPost, a.Url+"/api/login", bytes.NewBuffer(j))
	if err != nil {
		log.Println(err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := insecureClient.Do(req.WithContext(ctx))
	if err != nil {
		log.Println(err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return false
	}
	var r loginResp
	if err := json.Unmarshal(body, &r); err != nil {
		log.Println(err)
		return false
	}
	if r.Token == "" {
		return false
	}
	a.Token = r.Token
	return true
}

func (a *Api) loginFC(user, password string) bool {
	p := fcLoginParam{UserID: user, Password: password}
	j, err := json.Marshal(&p)
	if err != nil {
		log.Println(err)
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequest(http.MethodPost, a.Url+"/login", bytes.NewBuffer(j))
	if err != nil {
		log.Println(err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := insecureClient.Do(req.WithContext(ctx))
	if err != nil {
		log.Println(err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return false
	}
	var r loginResp
	if err := json.Unmarshal(body, &r); err != nil {
		log.Println(err)
		return false
	}
	if r.Token == "" {
		return false
	}
	a.Token = r.Token
	return true
}

type nodeEnt struct {
	Name  string `json:"Name"`
	State string `json:"State"`
}

func (a *Api) GetState() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequest(http.MethodGet, a.Url+"/api/nodes", nil)
	if err != nil {
		log.Println(err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+a.Token)
	resp, err := insecureClient.Do(req.WithContext(ctx))
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		a.Token = ""
		a.State = "unknown"
		return
	}
	if resp.StatusCode != http.StatusOK {
		a.State = "unknown"
		return
	}
	if resp.ContentLength > 1024*64*1024 {
		log.Printf("size over %d", resp.ContentLength)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}
	var nodes []nodeEnt
	if err := json.Unmarshal(body, &nodes); err != nil {
		log.Println(err)
		return
	}
	a.State = "unknown"
	for _, n := range nodes {
		switch n.State {
		case "high":
			a.State = "high"
			return
		case "low":
			a.State = "low"
		case "warn":
			if a.State != "low" {
				a.State = "warn"
			}
		case "normal", "repair":
			if a.State == "unknown" {
				a.State = "normal"
			}
		}
	}
}
