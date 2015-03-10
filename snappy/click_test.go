package snappy

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"launchpad.net/snappy/helpers"

	. "launchpad.net/gocheck"
)

func (s *SnapTestSuite) TestReadManifest(c *C) {
	manifestData := []byte(`{
   "description": "This is a simple hello world example.",
    "framework": "ubuntu-core-15.04-dev1",
    "hooks": {
        "echo": {
            "apparmor": "meta/echo.apparmor",
            "bin-path": "bin/echo"
        },
        "env": {
            "apparmor": "meta/env.apparmor",
            "bin-path": "bin/env"
        },
        "evil": {
            "apparmor": "meta/evil.apparmor",
            "bin-path": "bin/evil"
        }
    },
    "icon": "meta/hello.svg",
    "installed-size": "59",
    "maintainer": "Michael Vogt <mvo@ubuntu.com>",
    "name": "hello-world",
    "title": "Hello world example",
    "version": "1.0.5"
}`)
	manifest, err := readClickManifest(manifestData)
	c.Assert(err, IsNil)
	c.Assert(manifest.Name, Equals, "hello-world")
	c.Assert(manifest.Version, Equals, "1.0.5")
	c.Assert(manifest.Hooks["evil"]["bin-path"], Equals, "bin/evil")
	c.Assert(manifest.Hooks["evil"]["apparmor"], Equals, "meta/evil.apparmor")
}

func makeClickHook(c *C, hooksDir, hookName, hookContent string) {
	if _, err := os.Stat(hooksDir); err != nil {
		os.MkdirAll(hooksDir, 0755)
	}
	ioutil.WriteFile(path.Join(hooksDir, hookName+".hook"), []byte(hookContent), 0644)
}

func (s *SnapTestSuite) TestReadClickHookFile(c *C) {
	mockHooksDir := path.Join(s.tempdir, "hooks")
	makeClickHook(c, mockHooksDir, "snappy-systemd", `Hook-Name: systemd
User: root
Exec: /usr/lib/click-systemd/systemd-clickhook
Pattern: /var/lib/systemd/click/${id}`)
	hook, err := readClickHookFile(path.Join(mockHooksDir, "snappy-systemd.hook"))
	c.Assert(err, IsNil)
	c.Assert(hook.name, Equals, "systemd")
	c.Assert(hook.user, Equals, "root")
	c.Assert(hook.exec, Equals, "/usr/lib/click-systemd/systemd-clickhook")
	c.Assert(hook.pattern, Equals, "/var/lib/systemd/click/${id}")

	// click allows non-existing "Hook-Name" and uses the filename then
	makeClickHook(c, mockHooksDir, "apparmor", `
Pattern: /var/lib/apparmor/click/${id}`)
	hook, err = readClickHookFile(path.Join(mockHooksDir, "apparmor.hook"))
	c.Assert(err, IsNil)
	c.Assert(hook.name, Equals, "apparmor")
}

func (s *SnapTestSuite) TestReadClickHooksDir(c *C) {
	mockHooksDir := path.Join(s.tempdir, "hooks")
	makeClickHook(c, mockHooksDir, "snappy-systemd", `Hook-Name: systemd
User: root
Exec: /usr/lib/click-systemd/systemd-clickhook
Pattern: /var/lib/systemd/click/${id}`)
	clickSystemHooksDir = mockHooksDir
	hooks, err := systemClickHooks()
	c.Assert(err, IsNil)
	c.Assert(hooks, HasLen, 1)
	c.Assert(hooks["systemd"].name, Equals, "systemd")
}

func (s *SnapTestSuite) TestHandleClickHooks(c *C) {
	mockHooksDir := path.Join(s.tempdir, "hooks")

	// two hooks to ensure iterating works correct
	os.MkdirAll(path.Join(s.tempdir, "/var/lib/systemd/click/"), 0755)
	testSymlinkDir := path.Join(s.tempdir, "/var/lib/systemd/click/")
	content := fmt.Sprintf(`Hook-Name: systemd
Pattern: %s/${id}`, testSymlinkDir)
	makeClickHook(c, mockHooksDir, "snappy-systemd", content)

	os.MkdirAll(path.Join(s.tempdir, "/var/lib/apparmor/click/"), 0755)
	testSymlinkDir2 := path.Join(s.tempdir, "/var/lib/apparmor/click/")
	content = fmt.Sprintf(`Hook-Name: apparmor
Pattern: %s/${id}`, testSymlinkDir2)
	makeClickHook(c, mockHooksDir, "click-apparmor", content)

	instDir := path.Join(s.tempdir, "apps", "foo", "1.0")
	os.MkdirAll(instDir, 0755)
	ioutil.WriteFile(path.Join(instDir, "path-to-systemd-file"), []byte(""), 0644)
	ioutil.WriteFile(path.Join(instDir, "path-to-apparmor-file"), []byte(""), 0644)
	manifest := clickManifest{
		Name:    "foo",
		Version: "1.0",
		Hooks: map[string]clickAppHook{
			"app": clickAppHook{
				"systemd":  "path-to-systemd-file",
				"apparmor": "path-to-apparmor-file",
			},
		},
	}
	clickSystemHooksDir = mockHooksDir
	err := installClickHooks(instDir, manifest)
	c.Assert(err, IsNil)
	p := fmt.Sprintf("%s/%s_%s_%s", testSymlinkDir, manifest.Name, "app", manifest.Version)
	_, err = os.Stat(p)
	c.Assert(err, IsNil)
	symlinkTarget, err := filepath.EvalSymlinks(p)
	c.Assert(err, IsNil)
	c.Assert(symlinkTarget, Equals, path.Join(instDir, "path-to-systemd-file"))

	p = fmt.Sprintf("%s/%s_%s_%s", testSymlinkDir2, manifest.Name, "app", manifest.Version)
	_, err = os.Stat(p)
	c.Assert(err, IsNil)
	symlinkTarget, err = filepath.EvalSymlinks(p)
	c.Assert(err, IsNil)
	c.Assert(symlinkTarget, Equals, path.Join(instDir, "path-to-apparmor-file"))

	// now ensure we can remove
	clickSystemHooksDir = mockHooksDir
	err = removeClickHooks(manifest)
	c.Assert(err, IsNil)
	_, err = os.Stat(fmt.Sprintf("%s/%s_%s_%s", testSymlinkDir, manifest.Name, "app", manifest.Version))
	c.Assert(err, NotNil)
}

func (s *SnapTestSuite) TestLocalSnapInstall(c *C) {
	snapFile := makeTestSnapPackage(c, "")
	err := installClick(snapFile, 0)
	c.Assert(err, IsNil)

	baseDir := filepath.Join(snapAppsDir, "foo", "1.0")
	contentFile := filepath.Join(baseDir, "bin", "foo")
	content, err := ioutil.ReadFile(contentFile)
	c.Assert(err, IsNil)
	c.Assert(string(content), Equals, "#!/bin/sh\necho \"hello\"")

	// ensure we have the manifest too
	_, err = os.Stat(filepath.Join(baseDir, ".click", "info", "foo.manifest"))
	c.Assert(err, IsNil)

	// ensure we have the data dir
	_, err = os.Stat(path.Join(s.tempdir, "var", "lib", "apps", "foo", "1.0"))
	c.Assert(err, IsNil)

	// ensure we have the hashes
	snap := NewInstalledSnapPart(filepath.Join(baseDir, "meta", "package.yaml"))
	c.Assert(snap.Hash(), Not(Equals), "")
}

func (s *SnapTestSuite) TestLocalSnapInstallDebsigVerifyFails(c *C) {
	runDebsigVerify = func(snapFile string, allowUnauth bool) (err error) {
		return errors.New("something went wrong")
	}

	snapFile := makeTestSnapPackage(c, "")
	err := installClick(snapFile, 0)
	c.Assert(err, NotNil)

	contentFile := path.Join(s.tempdir, "apps", "foo", "1.0", "bin", "foo")
	_, err = os.Stat(contentFile)
	c.Assert(err, NotNil)
}

// ensure that the right parameters are passed to runDebsigVerify()
func (s *SnapTestSuite) TestLocalSnapInstallDebsigVerifyPassesUnauth(c *C) {
	var expectedUnauth bool
	runDebsigVerify = func(snapFile string, allowUnauth bool) (err error) {
		c.Assert(allowUnauth, Equals, expectedUnauth)
		return nil
	}

	expectedUnauth = true
	snapFile := makeTestSnapPackage(c, "")
	err := installClick(snapFile, AllowUnauthenticated)
	c.Assert(err, IsNil)

	expectedUnauth = false
	err = installClick(snapFile, 0)
	c.Assert(err, IsNil)
}

func (s *SnapTestSuite) TestSnapRemove(c *C) {
	targetDir := path.Join(s.tempdir, "apps")
	err := installClick(makeTestSnapPackage(c, ""), 0)
	c.Assert(err, IsNil)

	instDir := path.Join(targetDir, "foo", "1.0")
	_, err = os.Stat(instDir)
	c.Assert(err, IsNil)

	err = removeClick(instDir)
	c.Assert(err, IsNil)

	_, err = os.Stat(instDir)
	c.Assert(err, NotNil)
}

func (s *SnapTestSuite) TestLocalOemSnapInstall(c *C) {
	snapFile := makeTestSnapPackage(c, `name: foo
version: 1.0
type: oem
icon: foo.svg
vendor: Foo Bar <foo@example.com>`)
	err := installClick(snapFile, 0)
	c.Assert(err, IsNil)

	contentFile := path.Join(s.tempdir, "oem", "foo", "1.0", "bin", "foo")
	_, err = os.Stat(contentFile)
	c.Assert(err, IsNil)
	_, err = os.Stat(path.Join(s.tempdir, "oem", "foo", "1.0", ".click", "info", "foo.manifest"))
	c.Assert(err, IsNil)
}

func (s *SnapTestSuite) TestClickSetActive(c *C) {
	packageYaml := `name: foo
icon: foo.svg
vendor: Foo Bar <foo@example.com>
`
	snapFile := makeTestSnapPackage(c, packageYaml+"version: 1.0")
	c.Assert(installClick(snapFile, AllowUnauthenticated), IsNil)

	snapFile = makeTestSnapPackage(c, packageYaml+"version: 2.0")
	c.Assert(installClick(snapFile, AllowUnauthenticated), IsNil)

	// ensure v2 is active
	repo := NewLocalSnapRepository(filepath.Join(s.tempdir, "apps"))
	parts, err := repo.Installed()
	c.Assert(err, IsNil)
	c.Assert(parts, HasLen, 2)
	c.Assert(parts[0].Version(), Equals, "1.0")
	c.Assert(parts[0].IsActive(), Equals, false)
	c.Assert(parts[1].Version(), Equals, "2.0")
	c.Assert(parts[1].IsActive(), Equals, true)

	// set v1 active
	err = setActiveClick(parts[0].(*SnapPart).basedir)
	parts, err = repo.Installed()
	c.Assert(err, IsNil)
	c.Assert(parts[0].Version(), Equals, "1.0")
	c.Assert(parts[0].IsActive(), Equals, true)
	c.Assert(parts[1].Version(), Equals, "2.0")
	c.Assert(parts[1].IsActive(), Equals, false)

}

func (s *SnapTestSuite) TestClickCopyData(c *C) {
	snapDataHomeGlob = filepath.Join(s.tempdir, "home", "*", "apps")
	homeDir := filepath.Join(s.tempdir, "home", "user1", "apps")
	homeData := filepath.Join(homeDir, "foo", "1.0")
	err := helpers.EnsureDir(homeData, 0755)
	c.Assert(err, IsNil)

	packageYaml := `name: foo
icon: foo.svg
vendor: Foo Bar <foo@example.com>
`
	canaryData := []byte("ni ni ni")

	snapFile := makeTestSnapPackage(c, packageYaml+"version: 1.0")
	c.Assert(installClick(snapFile, AllowUnauthenticated), IsNil)
	canaryDataFile := filepath.Join(snapDataDir, "foo", "1.0", "canary.txt")
	err = ioutil.WriteFile(canaryDataFile, canaryData, 0644)
	c.Assert(err, IsNil)
	err = ioutil.WriteFile(filepath.Join(homeData, "canary.home"), canaryData, 0644)
	c.Assert(err, IsNil)

	snapFile = makeTestSnapPackage(c, packageYaml+"version: 2.0")
	c.Assert(installClick(snapFile, AllowUnauthenticated), IsNil)
	newCanaryDataFile := filepath.Join(snapDataDir, "foo", "2.0", "canary.txt")
	content, err := ioutil.ReadFile(newCanaryDataFile)
	c.Assert(err, IsNil)
	c.Assert(content, DeepEquals, canaryData)

	newHomeDataCanaryFile := filepath.Join(homeDir, "foo", "2.0", "canary.home")
	content, err = ioutil.ReadFile(newHomeDataCanaryFile)
	c.Assert(err, IsNil)
	c.Assert(content, DeepEquals, canaryData)
}

// ensure that even with no home dir there is no error and the
// system data gets copied
func (s *SnapTestSuite) TestClickCopyDataNoUserHomes(c *C) {
	// this home dir path does not exist
	snapDataHomeGlob = filepath.Join(s.tempdir, "no-such-home", "*", "apps")

	packageYaml := `name: foo
icon: foo.svg
vendor: Foo Bar <foo@example.com>
`
	snapFile := makeTestSnapPackage(c, packageYaml+"version: 1.0")
	c.Assert(installClick(snapFile, AllowUnauthenticated), IsNil)
	canaryDataFile := filepath.Join(snapDataDir, "foo", "1.0", "canary.txt")
	err := ioutil.WriteFile(canaryDataFile, []byte(""), 0644)

	snapFile = makeTestSnapPackage(c, packageYaml+"version: 2.0")
	c.Assert(installClick(snapFile, AllowUnauthenticated), IsNil)
	_, err = os.Stat(filepath.Join(snapDataDir, "foo", "2.0", "canary.txt"))
	c.Assert(err, IsNil)
}

const expectedWrapper = `#!/bin/sh
# !!!never remove this line!!!
##TARGET=/apps/pastebinit.mvo/1.4.0.0.1/bin/pastebinit

set -e

TMPDIR="/tmp/snapps/pastebinit.mvo/1.4.0.0.1/tmp"
if [ ! -d "$TMPDIR" ]; then
    mkdir -p -m1777 "$TMPDIR"
fi
export TMPDIR
export TEMPDIR="$TMPDIR"

# app paths (deprecated)
export SNAPP_APP_PATH="/apps/pastebinit.mvo/1.4.0.0.1/"
export SNAPP_APP_DATA_PATH="/var/lib//apps/pastebinit.mvo/1.4.0.0.1/"
export SNAPP_APP_USER_DATA_PATH="$HOME//apps/pastebinit.mvo/1.4.0.0.1/"
export SNAPP_APP_TMPDIR="$TMPDIR"
export SNAPP_OLD_PWD="$(pwd)"

# app paths
export SNAP_APP_PATH="/apps/pastebinit.mvo/1.4.0.0.1/"
export SNAP_APP_DATA_PATH="/var/lib//apps/pastebinit.mvo/1.4.0.0.1/"
export SNAP_APP_USER_DATA_PATH="$HOME//apps/pastebinit.mvo/1.4.0.0.1/"
export SNAP_APP_TMPDIR="$TMPDIR"

# FIXME: this will need to become snappy arch or something
export SNAPPY_APP_ARCH="$(dpkg --print-architecture)"

if [ ! -d "$SNAP_APP_USER_DATA_PATH" ]; then
   mkdir -p "$SNAP_APP_USER_DATA_PATH"
fi
export HOME="$SNAP_APP_USER_DATA_PATH"

# export old pwd
export SNAP_OLD_PWD="$(pwd)"
cd /apps/pastebinit.mvo/1.4.0.0.1/
aa-exec -p pastebinit.mvo_pastebinit_1.4.0.0.1 -- /apps/pastebinit.mvo/1.4.0.0.1/bin/pastebinit "$@"
`

func (s *SnapTestSuite) TestSnappyGenerateSnapBinaryWrapper(c *C) {
	binary := Binary{Name: "bin/pastebinit"}
	pkgPath := "/apps/pastebinit.mvo/1.4.0.0.1/"
	aaProfile := "pastebinit.mvo_pastebinit_1.4.0.0.1"
	m := packageYaml{Name: "pastebinit.mvo",
		Version: "1.4.0.0.1"}

	generatedWrapper := generateSnapBinaryWrapper(binary, pkgPath, aaProfile, &m)
	c.Assert(generatedWrapper, Equals, expectedWrapper)
}

func (s *SnapTestSuite) TestSnappyGetBinaryAaProfile(c *C) {
	m := packageYaml{Name: "foo",
		Version: "1.0"}

	c.Assert(getBinaryAaProfile(&m, Binary{Name: "bin/app"}), Equals, "foo_app_1.0")
	c.Assert(getBinaryAaProfile(&m, Binary{Name: "bin/app", SecurityTemplate: "some-security-json"}), Equals, "some-security-json")
	c.Assert(getBinaryAaProfile(&m, Binary{Name: "bin/app", SecurityPolicy: "some-profile"}), Equals, "some-profile")
}

func (s *SnapTestSuite) TestSnappyHandleBinariesOnInstall(c *C) {
	packageYaml := `name: foo.mvo
icon: foo.svg
vendor: Foo Bar <foo@example.com>
binaries:
 - name: bin/foo
`
	snapFile := makeTestSnapPackage(c, packageYaml+"version: 1.0")
	c.Assert(installClick(snapFile, AllowUnauthenticated), IsNil)

	// ensure that the binary wrapper file go generated with the right
	// name
	binaryWrapper := filepath.Join(snapBinariesDir, "foo.foo.mvo")
	c.Assert(helpers.FileExists(binaryWrapper), Equals, true)

	// and that it gets removed on remove
	snapDir := filepath.Join(snapAppsDir, "foo.mvo", "1.0")
	err := removeClick(snapDir)
	c.Assert(err, IsNil)
	c.Assert(helpers.FileExists(binaryWrapper), Equals, false)
	c.Assert(helpers.FileExists(snapDir), Equals, false)
}

func (s *SnapTestSuite) TestSnappyHandleBinariesOnUpgrade(c *C) {
	packageYaml := `name: foo.mvo
icon: foo.svg
vendor: Foo Bar <foo@example.com>
binaries:
 - name: bin/foo
`
	snapFile := makeTestSnapPackage(c, packageYaml+"version: 1.0")
	c.Assert(installClick(snapFile, AllowUnauthenticated), IsNil)

	// ensure that the binary wrapper file go generated with the right
	// path
	oldSnapBin := filepath.Join(snapAppsDir, "foo.mvo", "1.0", "bin", "foo")
	binaryWrapper := filepath.Join(snapBinariesDir, "foo.foo.mvo")
	content, err := ioutil.ReadFile(binaryWrapper)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(string(content), oldSnapBin), Equals, true)

	// and that it gets updated on upgrade
	snapFile = makeTestSnapPackage(c, packageYaml+"version: 2.0")
	c.Assert(installClick(snapFile, AllowUnauthenticated), IsNil)
	newSnapBin := filepath.Join(snapAppsDir, "foo.mvo", "2.0", "bin", "foo")
	content, err = ioutil.ReadFile(binaryWrapper)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(string(content), newSnapBin), Equals, true)
}

func (s *SnapTestSuite) TestSnappyHandleServicesOnInstall(c *C) {
	packageYaml := `name: foo.mvo
icon: foo.svg
vendor: Foo Bar <foo@example.com>
services:
 - name: service
   start: bin/hello
`
	snapFile := makeTestSnapPackage(c, packageYaml+"version: 1.0")
	c.Assert(installClick(snapFile, AllowUnauthenticated), IsNil)

	servicesFile := filepath.Join(snapServicesDir, "foo.mvo_service_1.0.service")
	c.Assert(helpers.FileExists(servicesFile), Equals, true)

	// and that it gets removed on remove
	snapDir := filepath.Join(snapAppsDir, "foo.mvo", "1.0")
	err := removeClick(snapDir)
	c.Assert(err, IsNil)
	c.Assert(helpers.FileExists(servicesFile), Equals, false)
	c.Assert(helpers.FileExists(snapDir), Equals, false)
}

const expectedService = `[Unit]
Description=The docker app deployment mechanism
After=apparmor.service
Requires=apparmor.service
X-Snappy=yes

[Service]
ExecStart=/apps/docker/1.3.3.001/bin/docker.wrap
WorkingDirectory=/apps/docker/1.3.3.001/
Environment="SNAPP_APP_PATH=/apps/docker/1.3.3.001/" "SNAPP_APP_DATA_PATH=/var/lib/apps/docker/1.3.3.001/" "SNAPP_APP_USER_DATA_PATH=%h/apps/docker/1.3.3.001/" "SNAP_APP_PATH=/apps/docker/1.3.3.001/" "SNAP_APP_DATA_PATH=/var/lib/apps/docker/1.3.3.001/" "SNAP_APP_USER_DATA_PATH=%h/apps/docker/1.3.3.001/" "SNAP_APP=docker_docker_1.3.3.001"
AppArmorProfile=docker_docker_1.3.3.001




[Install]
WantedBy=multi-user.target
`

func (s *SnapTestSuite) TestSnappyGenerateSnapServicesFile(c *C) {
	service := Service{Name: "docker",
		Start:       "bin/docker.wrap",
		Description: "The docker app deployment mechanism",
	}
	pkgPath := "/apps/docker/1.3.3.001/"
	aaProfile := "docker_docker_1.3.3.001"
	m := packageYaml{Name: "docker",
		Version: "1.3.3.001",
	}

	generated := generateSnapServicesFile(service, pkgPath, aaProfile, &m)
	c.Assert(generated, Equals, expectedService)
}