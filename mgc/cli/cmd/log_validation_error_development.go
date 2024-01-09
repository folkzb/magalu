//go:build !release

package cmd

func logValidationErr(e error) {
	logger().Warn("result validation failed", "error", e.Error())
}
