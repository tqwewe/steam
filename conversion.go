package steam

import (
	"math/big"
	"strconv"
	"strings"
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

// SteamIDToSteamID3 converts a given SteamID to a SteamID3.
// eg. STEAM_0:0:86173181 -> [U:1:172346362]
//
// An empty SteamID3 (string) is returned if the process was unsuccessful.
func SteamIDToSteamID3(steamID SteamID) SteamID3 {
	steamIDParts := strings.Split(string(steamID), ":")
	steamLastPart, err := strconv.ParseUint(string(steamIDParts[len(steamIDParts)-1]), 10, 64)
	if err != nil {
		return SteamID3("")
	}

	return SteamID3("[U:1:" + strconv.FormatUint(steamLastPart*2, 10) + "]")
}

// SteamID64ToSteamID converts a given SteamID64 to a SteamID.
// eg. 76561198132612090 -> STEAM_0:0:86173181
//
// An empty SteamID (string) is returned if the process was unsuccessful.
func SteamID64ToSteamID(steam64 SteamID64) SteamID {
	steamID := new(big.Int).SetInt64(int64(steam64))
	magic, _ := new(big.Int).SetString("76561197960265728", 10)
	steamID = steamID.Sub(steamID, magic)
	isServer := new(big.Int).And(steamID, big.NewInt(1))
	steamID = steamID.Sub(steamID, isServer)
	steamID = steamID.Div(steamID, big.NewInt(2))
	return SteamID("STEAM_0:" + isServer.String() + ":" + steamID.String())
}

// SteamID64ToSteamID32 converts a given SteamID64 to a SteamID32.
// eg. 76561198132612090 -> 172346362
//
// 0 is returned if the process was unsuccessful.
func SteamID64ToSteamID32(steam64 SteamID64) SteamID32 {
	steam64Str := strconv.FormatUint(uint64(steam64), 10)
	if len(steam64Str) < 3 {
		return 0
	}
	steam32, err := strconv.ParseInt(steam64Str[3:], 10, 64)
	if err != nil {
		return 0
	}
	return SteamID32(steam32 - 61197960265728)
}

// SteamID64ToSteamID3 converts a given SteamID64 to a SteamID3.
// eg. 76561198132612090 -> [U:1:172346362]
//
// An empty SteamID3 (string) is returned if the process was unsuccessful.
func SteamID64ToSteamID3(steam64 SteamID64) SteamID3 {
	steamID := SteamID64ToSteamID(steam64)
	if steamID == SteamID(0) {
		return SteamID3("")
	}

	return SteamIDToSteamID3(steamID)
}

// SteamID32ToSteamID converts a given SteamID32 to a SteamID.
// eg. 172346362 -> STEAM_0:0:86173181
//
// An empty SteamID (string) is returned if the process was unsuccessful.
func SteamID32ToSteamID(steam32 SteamID32) SteamID {
	return SteamID64ToSteamID(SteamID32ToSteamID64(steam32))
}

// SteamID32ToSteamID64 converts a given SteamID32 to a SteamID64.
// eg. 172346362 -> 76561198132612090
//
// 0 is returned if the process was unsuccessful.
func SteamID32ToSteamID64(steam32 SteamID32) SteamID64 {
	steam64, err := strconv.ParseInt("765"+strconv.FormatInt(int64(steam32)+61197960265728, 10), 10, 64)
	if err != nil {
		return 0
	}
	return SteamID64(steam64)
}

// SteamID32ToSteamID3 converts a given SteamID32 to a SteamID3.
// eg. 172346362 -> [U:1:172346362]
//
// An empty SteamID3 (string) is returned if the process was unsuccessful.
func SteamID32ToSteamID3(steam32 SteamID32) SteamID3 {
	steamID := SteamID32ToSteamID(steam32)
	if steamID == SteamID(0) {
		return SteamID3("")
	}

	return SteamIDToSteamID3(steamID)
}

// SteamID3ToSteamID converts a given SteamID3 to a SteamID.
// eg. [U:1:172346362] -> STEAM_0:0:86173181
//
// An empty SteamID (string) is returned if the process was unsuccessful.
func SteamID3ToSteamID(steam3 SteamID3) SteamID {
	IDparts := strings.Split(string(steam3), ":")

	id32 := IDparts[len(IDparts)-1]

	if len(id32) <= 0 {
		return SteamID("")
	}

	if id32[len(id32)-1:] == "]" {
		id32 = id32[:len(id32)-1]
	}

	steam32, err := strconv.ParseUint(id32, 10, 64)
	if err != nil {
		return SteamID("")
	}

	return SteamID32ToSteamID(SteamID32(steam32))
}

// SteamID3ToSteamID64 converts a given SteamID3 to a SteamID64.
// eg. [U:1:172346362] -> 76561198132612090
//
// 0 is returned if the process was unsuccessful.
func SteamID3ToSteamID64(steam3 SteamID3) SteamID64 {
	IDparts := strings.Split(string(steam3), ":")

	id32 := IDparts[len(IDparts)-1]

	if len(id32) <= 0 {
		return SteamID64(0)
	}

	if id32[len(id32)-1:] == "]" {
		id32 = id32[:len(id32)-1]
	}

	steam32, err := strconv.ParseUint(id32, 10, 64)
	if err != nil {
		return SteamID64(0)
	}

	return SteamID32ToSteamID64(SteamID32(steam32))
}

// SteamID3ToSteamID64 converts a given SteamID3 to a SteamID64.
// eg. [U:1:172346362] -> 172346362
//
// 0 is returned if the process was unsuccessful.
func SteamID3ToSteamID32(steam3 SteamID3) SteamID32 {
	IDparts := strings.Split(string(steam3), ":")

	id32 := IDparts[len(IDparts)-1]

	if len(id32) <= 0 {
		return SteamID32(0)
	}

	if id32[len(id32)-1:] == "]" {
		id32 = id32[:len(id32)-1]
	}

	steam32, err := strconv.ParseUint(id32, 10, 64)
	if err != nil {
		return SteamID32(0)
	}

	return SteamID32(steam32)
}
