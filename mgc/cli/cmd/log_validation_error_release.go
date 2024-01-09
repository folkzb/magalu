//go:build release

package cmd

func logValidationErr(e error) {
	logger().Debugw("result validation failed", "error", e.Error())
}
