package steam

import (
	"strings"
	"math/big"
	"strconv"
)

// SteamIDToSteamID64 converts a given SteamID to a SteamID64.
// eg. STEAM_0:0:86173181 -> 76561198132612090
//
// 0 is returned if the process was unsuccessful.
func SteamIDToSteamID64(steamID SteamID) SteamID64 {
	idParts := strings.Split(string(steamID), ":")
	magic, _ := new(big.Int).SetString("76561197960265728", 10)
	steam64, _ := new(big.Int).SetString(idParts[2], 10)
	steam64 = steam64.Mul(steam64, big.NewInt(2))
	steam64 = steam64.Add(steam64, magic)
	auth, _ := new(big.Int).SetString(idParts[1], 10)
	return SteamID64(steam64.Add(steam64, auth).Int64())
}

// SteamIDToSteamID32 converts a given SteamID to a SteamID32.
// eg. STEAM_0:0:86173181 -> 172346362
//
// 0 is returned if the process was unsuccessful.
func SteamIDToSteamID32(steamID SteamID) SteamID32 {
	return SteamID64ToSteamID32(SteamIDToSteamID64(steamID))
}

// SteamIDToSteamID32 converts a given SteamID to a SteamID3.
// eg. STEAM_0:0:86173181 -> [U:1:172346362]
//
// An empty SteamID3 (string) is returned if the process was unsuccessful.
/*func SteamIDToSteamID3(steamID SteamID) SteamID3 {
	return ""
}*/

// SteamID64ToSteamID32 converts a given SteamID64 to a SteamID32.
// eg. 76561198132612090 -> 172346362
//
// 0 is returned if the process was unsuccessful.
func SteamID64ToSteamID32(steam64 SteamID64) SteamID32 {
	steam32, err := strconv.ParseInt(strconv.FormatUint(uint64(steam64), 10)[3:], 10, 64)
	if err != nil {
		return 0
	}
	return SteamID32(steam32 - 61197960265728)
}

// SteamID32ToSteamID64 converts a given SteamID32 to a SteamID64.
// eg. 172346362 -> 76561198132612090
//
// 0 is returned if the process was unsuccessful.
func SteamID32ToSteamID64(steam32 SteamID32) SteamID64 {
	steam64, err := strconv.ParseInt("765" + strconv.FormatInt(int64(steam32) + 61197960265728, 10), 10, 64)
	if err != nil {
		return 0
	}
	return SteamID64(steam64)
}

// SteamID64ToSteamID converts a given SteamID64 to a SteamID.
// eg. 76561198132612090 -> STEAM_0:0:86173181
func SteamID64ToSteamID(steam64 SteamID64) SteamID {
	steamID := new(big.Int).SetInt64(int64(steam64))
	magic, _ := new(big.Int).SetString("76561197960265728", 10)
	steamID = steamID.Sub(steamID, magic)
	isServer := new(big.Int).And(steamID, big.NewInt(1))
	steamID = steamID.Sub(steamID, isServer)
	steamID = steamID.Div(steamID, big.NewInt(2))
	return SteamID("STEAM_0:" + isServer.String() + ":" + steamID.String())
}

// SteamID32ToSteamID converts a given SteamID32 to a SteamID.
// eg. 172346362 -> STEAM_0:0:86173181
func SteamID32ToSteamID(steam32 SteamID32) SteamID {
	return SteamID64ToSteamID(SteamID32ToSteamID64(steam32))
}