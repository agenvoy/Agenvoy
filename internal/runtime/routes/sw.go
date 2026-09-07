package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/pardnchiu/agenvoy/page"
)

var swSkip = []string{"icon.icns", "icon.ico"}

const swBody = `const CACHE = "agenvoy-" + VERSION;
const ASSETS = new Set(PRECACHE.concat(VENDOR));

self.addEventListener("install", function (event) {
  event.waitUntil(
    caches
      .open(CACHE)
      .then(function (cache) {
        return cache.addAll(PRECACHE).then(function () {
          return Promise.all(
            VENDOR.map(function (url) {
              return cache.add(url).catch(function () {});
            })
          );
        });
      })
      .then(function () {
        return self.skipWaiting();
      })
  );
});

self.addEventListener("activate", function (event) {
  event.waitUntil(
    caches
      .keys()
      .then(function (keys) {
        return Promise.all(
          keys.map(function (key) {
            return key === CACHE ? null : caches.delete(key);
          })
        );
      })
      .then(function () {
        return self.clients.claim();
      })
  );
});

function fromCache(key, request) {
  return caches.open(CACHE).then(function (cache) {
    return cache.match(key).then(function (hit) {
      if (hit) {
        return hit;
      }
      return fetch(request).then(function (response) {
        if (response.ok) {
          cache.put(key, response.clone());
        }
        return response;
      });
    });
  });
}

self.addEventListener("fetch", function (event) {
  const request = event.request;
  if (request.method !== "GET") {
    return;
  }
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) {
    return;
  }
  if (request.mode === "navigate") {
    event.respondWith(fromCache("/", request));
    return;
  }
  if (ASSETS.has(url.pathname)) {
    event.respondWith(fromCache(url.pathname, request));
  }
});
`

func swWorker() (string, error) {
	raw, err := page.FS.ReadFile("index.html")
	if err != nil {
		return "", fmt.Errorf("page.FS.ReadFile: %w", err)
	}
	stamp := map[string]string{"/": swSum(raw)}

	err = fs.WalkDir(page.FS, "public", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		base := strings.TrimPrefix(name, "public/")
		if slices.Contains(swSkip, base) || strings.HasPrefix(path.Base(name), ".") {
			return nil
		}
		content, err := page.FS.ReadFile(name)
		if err != nil {
			return err
		}
		stamp["/public/"+base] = swSum(content)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("fs.WalkDir: %w", err)
	}
	precache := slices.Sorted(maps.Keys(stamp))

	manifest, err := page.FS.ReadFile("vendor.json")
	if err != nil {
		return "", fmt.Errorf("page.FS.ReadFile: %w", err)
	}
	var list []struct {
		Path string `json:"path"`
		URL  string `json:"url"`
	}
	if err := json.Unmarshal(manifest, &list); err != nil {
		return "", fmt.Errorf("json.Unmarshal: %w", err)
	}

	vendor := make([]string, 0, len(list))
	for _, one := range list {
		url := "/vendor/" + one.Path
		vendor = append(vendor, url)
		stamp[url] = one.URL
	}
	slices.Sort(vendor)

	digest := sha256.New()
	for _, url := range slices.Sorted(maps.Keys(stamp)) {
		fmt.Fprintf(digest, "%s:%s\n", url, stamp[url])
	}
	version := hex.EncodeToString(digest.Sum(nil))[:12]

	var out strings.Builder
	fmt.Fprintf(&out, "const VERSION = %q;\n", version)
	fmt.Fprintf(&out, "const PRECACHE = %s;\n", swList(precache))
	fmt.Fprintf(&out, "const VENDOR = %s;\n", swList(vendor))
	out.WriteString(swBody)
	return out.String(), nil
}

func swSum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func swList(list []string) string {
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(raw)
}
