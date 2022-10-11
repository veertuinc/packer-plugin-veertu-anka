package util

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"bytes"

	"github.com/groob/plist"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/pathing"
)

var (
	random  *rand.Rand
	letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

// InstallerAppPlist is a list of variables that comes from the installer app
type InstallerAppPlist struct {
	OSVersion      string `plist:"DTPlatformVersion"`
	BundlerVersion string `plist:"CFBundleShortVersionString"`
}

// InstallerIPSWPlist is a list of variables that comes from the ipsw
type InstallerIPSWPlist struct {
	ProductVersion      string `plist:"ProductVersion"`
	ProductBuildVersion string `plist:"ProductBuildVersion"`
}

// Util defines everything this utility can do
type Util interface {
	ConfigTmpDir() (string, error)
	ConvertDiskSizeToBytes(diskSize string) (uint64, error)
	ObtainMacOSVersionFromInstallerApp(path string) (InstallerAppPlist, error)
	ObtainMacOSVersionFromInstallerIPSW(path string) (InstallerIPSWPlist, error)
	RandSeq(n int) string
	StepError(ui packer.Ui, state multistep.StateBag, err error) multistep.StepAction
	// ExecuteHostCommand(name string, arg ...string) string
}

// AnkaUtil implements Util
type AnkaUtil struct {
}

// StepError will return a halt action when any step fails
func (u *AnkaUtil) StepError(ui packer.Ui, state multistep.StateBag, err error) multistep.StepAction {
	state.Put("error", err)

	ui.Error(err.Error())

	return multistep.ActionHalt
}

// ConvertDiskSizeToBytes will convert the string into actual bytes the vm utilizes
func (u *AnkaUtil) ConvertDiskSizeToBytes(diskSize string) (uint64, error) {
	match, err := regexp.MatchString("^[0-9]+[g|G|m|M]$", diskSize)
	if err != nil {
		return uint64(0), err
	}
	if !match {
		return 0, fmt.Errorf("Input %s is not a valid disk size input", diskSize)
	}

	numericValue, err := strconv.Atoi(diskSize[:len(diskSize)-1])
	if err != nil {
		return uint64(0), err
	}
	suffix := diskSize[len(diskSize)-1:]

	switch strings.ToUpper(suffix) {
	case "G":
		return uint64(numericValue * 1024 * 1024 * 1024), nil
	case "M":
		return uint64(numericValue * 1024 * 1024), nil
	default:
		return uint64(0), fmt.Errorf("Invalid disk size suffix: %s", suffix)
	}
}

// ObtainMacOSVersionFromInstaller abstracts the os version from the installer ipsw provided
func (u *AnkaUtil) ObtainMacOSVersionFromInstallerIPSW(path string) (InstallerIPSWPlist, error) {
	installerPlist := InstallerIPSWPlist{}
	plistContent := bytes.NewReader([]byte(executeHostCommand("unzip", "-p", path, "SystemVersion.plist")))
	err := plist.NewXMLDecoder(plistContent).Decode(&installerPlist)
	if err != nil {
		return installerPlist, err
	}
	return installerPlist, nil
}

// ObtainMacOSVersionFromInstaller abstracts the os version from the installer app provided
func (u *AnkaUtil) ObtainMacOSVersionFromInstallerApp(path string) (InstallerAppPlist, error) {
	installerPlist := InstallerAppPlist{}
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return installerPlist, fmt.Errorf("installer app does not exist at %q: %w", path, err)
	}
	if err != nil {
		return installerPlist, fmt.Errorf("failed to stat installer at %q: %w", path, err)
	}
	plistPath := filepath.Join(path, "Contents", "Info.plist")	
	_, err = os.Stat(plistPath)
	if os.IsNotExist(err) {
		return installerPlist, fmt.Errorf("installer app info plist did not exist at %q: %w", plistPath, err)
	}
	if err != nil {
		return installerPlist, fmt.Errorf("failed to stat installer app info plist at %q: %w", plistPath, err)
	}
	plistContent, _ := os.Open(plistPath)
	err = plist.NewXMLDecoder(plistContent).Decode(&installerPlist)
	if err != nil {
		return installerPlist, err
	}
	return installerPlist, nil
}

// ConfigTmpDir creates the temp dir used by packer during runtime
func (u *AnkaUtil) ConfigTmpDir() (string, error) {
	configdir, err := pathing.ConfigDir()
	if err != nil {
		return "", err
	}

	tmpdir := os.Getenv("PACKER_TMP_DIR")
	if tmpdir != "" {
		fp, err := filepath.Abs(tmpdir)
		log.Printf("found PACKER_TMP_DIR env variable; setting tmpdir to %s", fp)
		if err != nil {
			return "", err
		}

		configdir = fp
	}

	_, err = os.Stat(configdir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Config dir %s does not exist; creating...", configdir)

			err = os.MkdirAll(configdir, 0755)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	td, err := ioutil.TempDir(configdir, "tmp")
	if err != nil {
		return "", fmt.Errorf("Error creating temp dir: %s", err)
	}

	log.Printf("Set Packer temp dir to %s", td)
	return td, nil
}

func (u *AnkaUtil) RandSeq(n int) string {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[random.Intn(len(letters))]
	}

	return string(b)
}

func executeHostCommand(name string, arg ...string) string {
	output, _ := exec.Command(name, arg...).CombinedOutput()

	return string(output)
}
