# Go library to work with Broadlink devices

#### Power Control (SP2 or similar devices)
 - SP2 : 0x2711
 - Honeywell SP2 : 0x2719, 0x7919, 0x271a, 0x791a 
 - SPMini : 0x2720
 - SP3 : 0x753e
 - OEM branded SP3 : 0x7D00
 - SP3S : 0x947a, 0x9479
 - SPMini2 : 0x2728
 - OEM branded SPMini : 0x2733, 0x273e
 - OEM branded SPMini2 : 0x7530, 0x7546, 0x7918
 - TMall OEM SPMini3 : 0x7D0D
 - SPMiniPlus : 0x2736
```
package main

import (
  "fmt"
  "github.com/sayzard/broadlinkzard"
)

func main() {

  dev := broadlinkzard.NewBroadlinkDirectDevice(0x947a, "192.168.0.xxx", "34:ea:34:xx:xx:xx")
  defer dev.Close()
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

#### Power Control (MP1 or similar devices)
 - MP1 : 0x2711
 - Honyar oem mp1 : 0x4EF7
 ```
package main

import (
  "fmt"
  "github.com/sayzard/broadlinkzard"
)

func main() {

  dev := broadlinkzard.NewBroadlinkDirectDevice(0x4EB5, "192.168.0.xxx", "34:ea:34:xx:xx:xx")
  defer dev.Close()
  dev.SetLogLevel(0)
  _, err := dev.Auth()
  if err != nil {
    panic(err)
  }
  _, err = dev.SetPowerMulti(1,true) // on - true , off - false
  if err != nil {
    panic(err)
  }
  
  pmask, err := dev.QueryPowerRaw()
  if err != nil {
    panic(err)
  }
  fmt.Println("Done",pmask)
}
```

### References
* <https://github.com/mjg59/python-broadlink>
