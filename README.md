# Go library to work with Broadlink devices

Power Control (SP2 or similar devices)
```
package main

import (
  "fmt"
  "github.com/sayzard/broadlinkzard"
)

func main() {

  dev := broadlinkzard.NewBroadlinkDirectDevice(0x947a, "192.168.0.xxx", "34:ea:34:xx:xx:xx")
  dev.SetLogLevel(0)
  _, err := dev.Auth()
  if err != nil {
    panic(err)
  }
  _, err = dev.SetPower(false) // on - true , off - false
  if err != nil {
    panic(err)
  }
  fmt.Println("Done")
}
```

### References
* <https://github.com/mjg59/python-broadlink>
