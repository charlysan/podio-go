package podio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	httpClient *http.Client
	authToken  *AuthToken
}

type Organization struct {
	Id   uint   `json:"org_id"`
	Slug string `json:"url_label"`
	Name string `json:"name"`
}

type Space struct {
	Id   uint   `json:"space_id"`
	Slug string `json:"url_label"`
	Name string `json:"name"`
}

type App struct {
	Id   uint   `json:"app_id"`
	Name string `json:"name"`
}

type Item struct {
	Id                 uint     `json:"item_id"`
	AppItemId          uint     `json:"app_item_id"`
	FormattedAppItemId string   `json:"app_item_id_formatted"`
	Title              string   `json:"title"`
	Files              []*File  `json:"files"`
	Fields             []*Field `json:"fields"`
}

type Field struct {
	FieldID    uint     `json:"field_id"`
	ExternalID string   `json:"external_id"`
	Type       string   `json:"type"`
	Label      string   `json:"label"`
	Values     []*Value `json:"values"`
}

type Value struct {
	Value interface{} `json:"value"`
}

type ItemList struct {
	Filtered uint    `json:"filtered"`
	Total    uint    `json:"total"`
	Items    []*Item `json:"items"`
}

type File struct {
	Id   uint   `json:"file_id"`
	Name string `json:"name"`
	Link string `json:"link"`
	Size int    `json:"size"`
}

type Comment struct {
	ID         uint                   `json:"comment_id"`
	Value      string                 `json:"value"`
	Ref        map[string]interface{} `json:"ref"`
	Files      []*File                `json:"files"`
	CreatedBy  interface{}            `json:"created_by"`
	CreatedVia interface{}            `json:"created_via"`
	CreatedOn  interface{}            `json:"created_on"`
	IsLiked    bool                   `json:"is_liked"`
	LikeCount  int                    `json:"like_count"`
}

type AuthToken struct {
	AccessToken   string                 `json:"access_token"`
	TokenType     string                 `json:"token_type"`
	ExpiresIn     int                    `json:"expires_in"`
	RefreshToken  string                 `json:"refresh_token"`
	Ref           map[string]interface{} `json:"ref"`
	TransferToken string                 `json:"transfer_token"`
}

type Error struct {
	Parameters interface{} `json:"error_parameters"`
	Detail     interface{} `json:"error_detail"`
	Propagate  bool        `json:"error_propagate"`
	Request    struct {
		URL   string `json:"url"`
		Query string `json:"query_string"`
	} `json:"request"`
	Description string `json:"error_description"`
	Type        string `json:"error"`
}

func (p *Error) Error() string {
	return fmt.Sprintf("%s: %s", p.Type, p.Description)
}

func AuthWithUserCredentials(client_id string, client_secret string, username string, password string) (*AuthToken, error) {
	var authToken AuthToken

	data := url.Values{
		"grant_type":    {"password"},
		"username":      {username},
		"password":      {password},
		"client_id":     {client_id},
		"client_secret": {client_secret},
	}
	resp, err := http.PostForm("https://api.podio.com/oauth/token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(respBody, &authToken)
	if err != nil {
		return nil, err
	}

	return &authToken, nil
}

func AuthWithAppCredentials(client_id, client_secret string, app_id uint, app_token string) (*AuthToken, error) {
	var authToken AuthToken

	data := url.Values{
		"grant_type":    {"app"},
		"app_id":        {fmt.Sprintf("%d", app_id)},
		"app_token":     {app_token},
		"client_id":     {client_id},
		"client_secret": {client_secret},
	}
	resp, err := http.PostForm("https://api.podio.com/oauth/token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(respBody, &authToken)
	if err != nil {
		return nil, err
	}

	return &authToken, nil
}

func NewClient(authToken *AuthToken) *Client {
	return &Client{
		httpClient: &http.Client{},
		authToken:  authToken,
	}
}

func (client *Client) request(method string, path string, headers map[string]string, body io.Reader, out interface{}) error {
	req, err := http.NewRequest(method, "https://api.podio.com"+path, body)

	if err != nil {
		return err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	req.Header.Add("Authorization", "OAuth2 "+client.authToken.AccessToken)
	resp, err := client.httpClient.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if !(200 <= resp.StatusCode && resp.StatusCode < 300) {
		podioErr := &Error{}
		err := json.Unmarshal(respBody, podioErr)
		if err != nil {
			return errors.New(string(respBody))
		}
		return podioErr
	}

	if out != nil {
		err = json.Unmarshal(respBody, out)
		if err != nil {
			return err
		}
	}

	return nil
}

func (client *Client) GetOrganizations() (orgs []Organization, err error) {
	err = client.request("GET", "/org", nil, nil, &orgs)
	return
}

func (client *Client) GetOrganization(id uint) (org *Organization, err error) {
	path := fmt.Sprintf("/org/%d", id)
	err = client.request("GET", path, nil, nil, &org)
	return
}

func (client *Client) GetOrganizationBySlug(slug string) (org *Organization, err error) {
	path := fmt.Sprintf("/org/url?org_slug=%s", slug)
	err = client.request("GET", path, nil, nil, &org)
	return
}

func (client *Client) GetSpaces(org_id uint) (spaces []Space, err error) {
	path := fmt.Sprintf("/org/%d/space", org_id)
	err = client.request("GET", path, nil, nil, &spaces)
	return
}

func (client *Client) GetSpace(id uint) (space *Space, err error) {
	path := fmt.Sprintf("/space/%d", id)
	err = client.request("GET", path, nil, nil, &space)
	return
}

func (client *Client) GetSpaceByOrgIdAndSlug(org_id uint, slug string) (space *Space, err error) {
	path := fmt.Sprintf("/space/org/%d/%s", org_id, slug)
	err = client.request("GET", path, nil, nil, &space)
	return
}

func (client *Client) GetApps(space_id uint) (apps []App, err error) {
	path := fmt.Sprintf("/app/space/%d?view=micro", space_id)
	err = client.request("GET", path, nil, nil, &apps)
	return
}

func (client *Client) GetApp(id uint) (app *App, err error) {
	path := fmt.Sprintf("/app/%d?view=micro", id)
	err = client.request("GET", path, nil, nil, &app)
	return
}

func (client *Client) GetAppBySpaceIdAndSlug(space_id uint, slug string) (app *App, err error) {
	path := fmt.Sprintf("/app/space/%d/%s", space_id, slug)
	err = client.request("GET", path, nil, nil, &app)
	return
}

func (client *Client) GetItems(app_id uint) (items *ItemList, err error) {
	path := fmt.Sprintf("/item/app/%d/filter?fields=items.fields(files)", app_id)
	err = client.request("POST", path, nil, nil, &items)
	return
}

func (client *Client) GetItemByAppItemId(app_id uint, formatted_app_item_id string) (item *Item, err error) {
	path := fmt.Sprintf("/app/%d/item/%s", app_id, formatted_app_item_id)
	err = client.request("GET", path, nil, nil, &item)
	return
}

func (client *Client) GetItemByExternalID(app_id uint, external_id string) (item *Item, err error) {
	path := fmt.Sprintf("/item/app/%d/external_id/%s", app_id, external_id)
	err = client.request("GET", path, nil, nil, &item)
	return
}

func (client *Client) GetItem(item_id uint) (item *Item, err error) {
	path := fmt.Sprintf("/item/%d?fields=files", item_id)
	err = client.request("GET", path, nil, nil, &item)
	return
}

func (client *Client) CreateItem(app_id uint, external_id string, fieldValues map[string]interface{}) (uint, error) {
	path := fmt.Sprintf("/item/app/%d", app_id)
	val := map[string]interface{}{
		"fields": fieldValues,
	}

	if external_id != "" {
		val["external_id"] = external_id
	}

	buf, err := json.Marshal(val)
	if err != nil {
		return 0, err
	}

	rsp := &struct {
		ItemId uint `json:"item_id"`
	}{}
	err = client.request("POST", path, nil, bytes.NewReader(buf), rsp)

	return rsp.ItemId, err
}

func (client *Client) UpdateItem(itemId uint, fieldValues map[string]interface{}) error {
	path := fmt.Sprintf("/item/%d", itemId)
	buf, err := json.Marshal(map[string]interface{}{"fields": fieldValues})
	if err != nil {
		return err
	}

	return client.request("PUT", path, nil, bytes.NewBuffer(buf), nil)

}

func (client *Client) Comment(refType, refId, text string) (comment *Comment, err error) {
	path := fmt.Sprintf("/comment/%s/%d/", refType, refId)
	buf, err := json.Marshal(struct {
		Value string `json:"value"`
	}{text})
	if err != nil {
		return
	}

	err = client.request("POST", path, nil, bytes.NewReader(buf), comment)
	return
}

func (client *Client) GetComments(refType string, refId string) (comments []*Comment, err error) {
	path := fmt.Sprintf("/comment/%s/%s/", refType, refId)
	err = client.request("GET", path, nil, nil, &comments)
	return
}

func (client *Client) GetFiles() (files []File, err error) {
	err = client.request("GET", "/file", nil, nil, &files)
	return
}

func (client *Client) GetFile(file_id uint) (file *File, err error) {
	err = client.request("GET", fmt.Sprintf("/file/%d", file_id), nil, nil, &file)
	return
}

func (client *Client) GetFileContents(url string) ([]byte, error) {
	link := fmt.Sprintf("%s?oauth_token=%s", url, client.authToken.AccessToken)
	resp, err := http.Get(link)

	if err != nil {
		return nil, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func (client *Client) CreateFile(name string, contents []byte) (file *File, err error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("source", name)
	if err != nil {
		return nil, err
	}

	_, err = part.Write(contents)
	if err != nil {
		return nil, err
	}

	err = writer.WriteField("filename", name)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": writer.FormDataContentType(),
	}

	err = client.request("POST", "/file", headers, body, &file)
	return
}

func (client *Client) ReplaceFile(oldFileId, newFileId uint) error {
	path := fmt.Sprintf("/file/%d/replace", newFileId)
	body := strings.NewReader(fmt.Sprintf("{\"old_file_id\":%d}", oldFileId))
	return client.request("POST", path, nil, body, nil)
}

func (client *Client) AttachFile(fileId uint, refType string, refId uint) error {
	path := fmt.Sprintf("/file/%d/attach", fileId)
	body := strings.NewReader(fmt.Sprintf("{\"ref_type\":\"%s\",\"ref_id\":%d}", refType, refId))
	return client.request("POST", path, nil, body, nil)
}

func (client *Client) DeleteFile(fileId uint) error {
	path := fmt.Sprintf("/file/%d", fileId)
	return client.request("DELETE", path, nil, nil, nil)
}
