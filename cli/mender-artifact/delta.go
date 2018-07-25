
package main

import (
    "fmt"
    "os"

    "github.com/mendersoftware/mender-artifact/delta"
	"github.com/urfave/cli"
)


func validateInputDelta(c *cli.Context) error {
	if c.String("rootfs-old") == "" ||
        c.String("rootfs-new") == "" ||
        c.String("output-path") == "" {
		return cli.NewExitError(
			"must provide `rootfs-old`, `rootfs-new` and `output-path`",
			errArtifactInvalidParameters,
		)
	}
	return nil
}

func createDelta (c *cli.Context) error {
    if err := validateInputDelta(c); err != nil {
        Log.Error(err.Error())
        return err
    }

    fmt.Println(c.String("rootfs-old"))
    _src, err := os.OpenFile(c.String("rootfs-old"), os.O_RDONLY, 0664)
    if err != nil { return err}
    fmt.Println(c.String("rootfs-new"))
    _new, err := os.OpenFile(c.String("rootfs-new"), os.O_RDONLY, 0664)
    if err != nil { return err }
    _out, err := os.OpenFile(c.String("output-path"), os.O_WRONLY | os.O_CREATE, 0664)
    if err != nil { return err }

    buf := make([]byte, delta.ALLOCSIZE)
    dd := delta.NewDeltaDecoding(_src, _new, _out, buf)

    err = dd.Encode()
    if err != nil { return err }

    return nil
}

