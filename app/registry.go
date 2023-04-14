package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type Registry struct {
	authenticationServer string
	registryServer       string
	name                 string
	tag                  string
	chroot               string
}

func NewRegistry(imageName string, chroot string) *Registry {
	imageParts := strings.Split(imageName, ":")
	name := fmt.Sprintf("library/%s", imageParts[0])

	var tag string
	if len(imageParts) > 1 {
		tag = imageParts[1]
	} else {
		tag = "latest"
	}

	return &Registry{
		authenticationServer: "https://auth.docker.io/token?service=registry.docker.io",
		registryServer:       "https://registry.hub.docker.com/v2",
		name:                 name,
		tag:                  tag,
		chroot:               chroot,
	}
}

type Output struct {
	Manifest interface{}
}

func (r *Registry) Pull(ctx context.Context) error {
	token, err := r.getToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	layers, err := r.getLayers(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}

	err = r.pullLayers(ctx, token, layers)
	if err != nil {
		return fmt.Errorf("failed to pull layers: %w", err)
	}

	return nil
}

func (r *Registry) pullLayers(ctx context.Context, token string, layers []FsLayer) error {
	for _, layer := range layers {

		url, err := url.Parse(r.registryServer)
		if err != nil {
			return fmt.Errorf("failed to parse registry server url: %w", err)
		}
		url.Path = fmt.Sprintf("%s/%s/blobs/%s", url.Path, r.name, layer.BlobSum)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to get manifest: %w", err)
		}
		defer resp.Body.Close()

		tmpFile, err := ioutil.TempFile(r.chroot, "tmp_*.tar.gz")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		_, err = io.Copy(tmpFile, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to copy layer to temp file: %w", err)
		}

		err = tmpFile.Close()
		if err != nil {
			return fmt.Errorf("failed to close temp file: %w", err)
		}

		cmd := exec.Command("tar", "-xzf", tmpFile.Name(), "-C", r.chroot)
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to untar layer: %w", err)
		}

	}

	return nil
}

type FsLayer struct {
	BlobSum string `json:"blobSum,omitempty"`
}

type GetManifestResponse struct {
	SchemaVersion int       `json:"schemaVersion,omitempty"`
	Name          string    `json:"name,omitempty"`
	Tag           string    `json:"tag,omitempty"`
	Architecture  string    `json:"architecture,omitempty"`
	FsLayers      []FsLayer `json:"fsLayers,omitempty"`
}

func (r *Registry) getLayers(ctx context.Context, token string) ([]FsLayer, error) {
	url, err := url.Parse(r.registryServer)
	if err != nil {
		return []FsLayer{}, fmt.Errorf("failed to parse registry server url: %w", err)
	}
	url.Path = fmt.Sprintf("%s/%s/manifests/%s", url.Path, r.name, r.tag)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return []FsLayer{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []FsLayer{}, fmt.Errorf("failed to get manifest: %w", err)
	}
	var body GetManifestResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return []FsLayer{}, fmt.Errorf("failed to decode manifest response: %w", err)
	}

	return body.FsLayers, err
}

type GetTokenResponse struct {
	Token string
}

func (r *Registry) getToken(ctx context.Context) (string, error) {
	url, err := url.Parse(r.authenticationServer)
	if err != nil {
		return "", fmt.Errorf("failed to parse authentication server url: %w", err)
	}
	qValues := url.Query()
	qValues.Add("scope", fmt.Sprintf("repository:%s:pull", r.name))
	url.RawQuery = qValues.Encode()

	resp, err := http.Get(url.String())
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}
	var body GetTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return body.Token, nil
}
