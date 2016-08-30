# Steam ![alt text](http://i.picresize.com/images/2016/08/23/ROrL1.png "Steam")

Steam is a package individually created by Acidic9 for Golang. It provides a huge range of features and functions and was made to be used with ease.

Some major features include:
  * Convert SteamID's to nearly any format.
  * Log into Steam and send group/friend invites or message a friend.
  * A huge amount of Steam's API features compiled into individual functions.

### Installation

To download the package:
```ssh
go get github.com/Acidic9/steam
```

To import it into your document:
```php
import (
	"github.com/Acidic9/steam"
)
```

### Example Usage

__Convert a SteamID__
```go
package main

import (
	"fmt"
	"github.com/Acidic9/steam"
)

func main() {
	steam64 := steam.SteamID64(76561198132612090)
	fmt.Println("SteamID:   ", steam.SteamID64ToSteamID(steam64))
	fmt.Println("SteamID 64:", steam64)
	fmt.Println("SteamID 32:", steam.SteamID64ToSteamID32(steam64))
	fmt.Println("SteamID 3: ", steam.SteamID64ToSteamID3(steam64))
}
```

__Send a message to someone on Steam__
```go
package main

import (
	"github.com/Acidic9/steam"
	"log"
)

func main() {
	recipient := steam.SteamID64(76561198132612090)
	message := "Hello, I am sending this message through Golang's Steam package."

	acc, err := steam.Login("username", "password") // Cannot use an account with Steam Guard
	if err != nil {
		log.Fatal(err)
	}

	err = acc.Message(recipient, message)
	if err != nil {
		log.Fatal(err)
	}
}
```

---

### Official Godoc
The official Godoc for this package is available at [https://godoc.org/github.com/Acidic9/steam](https://godoc.org/github.com/Acidic9/steam)
