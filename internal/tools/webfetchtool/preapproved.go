package webfetchtool

import (
	"net/url"
	"strings"
)

// PREAPPROVED_ENTRIES mirrors preapproved.ts PREAPPROVED_HOSTS (hostname or host/path prefix).
var preapprovedEntries = []string{
	"platform.claude.com",
	"code.claude.com",
	"modelcontextprotocol.io",
	"github.com/anthropics",
	"agentskills.io",
	"docs.python.org",
	"en.cppreference.com",
	"docs.oracle.com",
	"learn.microsoft.com",
	"developer.mozilla.org",
	"go.dev",
	"pkg.go.dev",
	"www.php.net",
	"docs.swift.org",
	"kotlinlang.org",
	"ruby-doc.org",
	"doc.rust-lang.org",
	"www.typescriptlang.org",
	"react.dev",
	"angular.io",
	"vuejs.org",
	"nextjs.org",
	"expressjs.com",
	"nodejs.org",
	"bun.sh",
	"jquery.com",
	"getbootstrap.com",
	"tailwindcss.com",
	"d3js.org",
	"threejs.org",
	"redux.js.org",
	"webpack.js.org",
	"jestjs.io",
	"reactrouter.com",
	"docs.djangoproject.com",
	"flask.palletsprojects.com",
	"fastapi.tiangolo.com",
	"pandas.pydata.org",
	"numpy.org",
	"www.tensorflow.org",
	"pytorch.org",
	"scikit-learn.org",
	"matplotlib.org",
	"requests.readthedocs.io",
	"jupyter.org",
	"laravel.com",
	"symfony.com",
	"wordpress.org",
	"docs.spring.io",
	"hibernate.org",
	"tomcat.apache.org",
	"gradle.org",
	"maven.apache.org",
	"asp.net",
	"dotnet.microsoft.com",
	"nuget.org",
	"blazor.net",
	"reactnative.dev",
	"docs.flutter.dev",
	"developer.apple.com",
	"developer.android.com",
	"keras.io",
	"spark.apache.org",
	"huggingface.co",
	"www.kaggle.com",
	"www.mongodb.com",
	"redis.io",
	"www.postgresql.org",
	"dev.mysql.com",
	"www.sqlite.org",
	"graphql.org",
	"prisma.io",
	"docs.aws.amazon.com",
	"cloud.google.com",
	"kubernetes.io",
	"www.docker.com",
	"www.terraform.io",
	"www.ansible.com",
	"vercel.com/docs",
	"docs.netlify.com",
	"devcenter.heroku.com",
	"cypress.io",
	"selenium.dev",
	"docs.unity.com",
	"docs.unrealengine.com",
	"git-scm.com",
	"nginx.org",
	"httpd.apache.org",
}

var (
	preapprovedHostnameOnly map[string]struct{}
	preapprovedPathPrefixes map[string][]string
)

func init() {
	preapprovedHostnameOnly = make(map[string]struct{})
	preapprovedPathPrefixes = make(map[string][]string)
	for _, e := range preapprovedEntries {
		if i := strings.Index(e, "/"); i >= 0 {
			host := e[:i]
			path := e[i:]
			preapprovedPathPrefixes[host] = append(preapprovedPathPrefixes[host], path)
		} else {
			preapprovedHostnameOnly[e] = struct{}{}
		}
	}
}

// IsPreapprovedHost mirrors preapproved.ts isPreapprovedHost.
func IsPreapprovedHost(hostname, pathname string) bool {
	if pathname == "" {
		pathname = "/"
	}
	if _, ok := preapprovedHostnameOnly[hostname]; ok {
		return true
	}
	prefixes := preapprovedPathPrefixes[hostname]
	for _, p := range prefixes {
		if pathname == p || strings.HasPrefix(pathname, p+"/") {
			return true
		}
	}
	return false
}

// IsPreapprovedURL mirrors utils.ts isPreapprovedUrl.
func IsPreapprovedURL(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Hostname() == "" {
		return false
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	return IsPreapprovedHost(u.Hostname(), path)
}
