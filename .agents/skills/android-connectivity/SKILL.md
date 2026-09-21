---
name: android-connectivity
description: >
  This skill covers Android device-to-device connectivity: Wi-Fi Aware
  (NAN), Wi-Fi Direct (P2P), and satellite/NTN APIs, plus the
  NEARBY_WIFI_DEVICES and ACCESS_LOCAL_NETWORK permission model. Use it
  for API surfaces, discovery and data-path flows, pairing/security,
  permission requirements per API level, and what is actually usable by
  third-party apps vs carrier or system gated.
---

## When to use this skill

- You are building peer-to-peer Android features with Wi-Fi Aware or
  Wi-Fi Direct.
- You need the exact permission set for a targetSdk version (33, 34,
  36, 37).
- You are checking what Android's satellite APIs expose to third-party
  apps (very little).
- You are debugging discovery failures, SecurityExceptions, or missing
  MAC addresses.
- You are picking between Aware and Direct for a use case.

## How to use

1. Read this file for the capability map, the permission model, and the
   API-level matrix.
2. Load [references/wifi-aware.md](references/wifi-aware.md) for NAN
   discovery, messaging, data paths, pairing, and ranging.
3. Load [references/wifi-direct.md](references/wifi-direct.md) for the
   P2P flow, group model, R2 pairing, and service discovery.
4. Load [references/satellite.md](references/satellite.md) for the NTN
   stack and what is public vs `@SystemApi`.
5. Fall back to developer.android.com connectivity docs for
   class-level detail.

## Examples

- "What permissions does Wi-Fi Aware need on targetSdk 34 vs 37?"
- "Walk me through the Aware flow from attach to an open socket."
- "Why does onIdentityChanged give me an all-zeros MAC?"
- "Can my app send data over satellite on a Pixel?"
- "Set up a Wi-Fi Direct group with a known passphrase on API 36."

# Android device-to-device connectivity

Three mechanisms cover three different jobs:

| Mechanism | Role | Public since | Third-party usable |
|---|---|---|---|
| Wi-Fi Aware (NAN) | Discovery + small messages + negotiated data paths | API 26 | Yes |
| Wi-Fi Direct (P2P) | Group-forming connection, GO acts as DHCP AP | API 14 | Yes |
| Satellite (NTN) | Carrier satellite messaging/data | API 35-37 | Only state listening and constrained-data opt-in |

## Choosing

- **Wi-Fi Aware** when you need discovery, short messages, and a
  negotiated encrypted data path without a user pairing dialog.
  Discovery is multicast-style service publish/subscribe. Data paths
  are per-peer and can be PSK/PMK secured. No group owner, no DHCP.
- **Wi-Fi Direct** when you need a classic group with a group owner
  that hands out IP addresses, or when the peer is a legacy device
  (printer, camera) that only speaks P2P. One group at a time, and
  legacy pairing can surface a system WPS dialog.
- **Satellite** when you are on a carrier NTN plan. Third-party apps
  cannot send satellite data directly. They can listen for modem
  state (API 36) and opt into constrained satellite data as an
  "optimized" app (API 36.1). Everything else is `@SystemApi` +
  signature permission.

## Permission model

### NEARBY_WIFI_DEVICES (API 33)

`android.permission.NEARBY_WIFI_DEVICES` is a dangerous/runtime
permission in the `NEARBY_DEVICES` group (with `BLUETOOTH_*`,
`UWB_RANGING`, and `ACCESS_LOCAL_NETWORK`). One dialog covers the
group.

Required on targetSdk 33+ for: Aware attach/discovery/data paths,
P2P discovery/connect, RTT to Aware peers, local-only hotspot.

The `android:usesPermissionFlags="neverForLocation"` flag asserts no
location derivation. Two consequences:

- Without the flag, the system **additionally requires
  `ACCESS_FINE_LOCATION`** for the same operations.
- With the flag, identity stays private: `onIdentityChanged`
  (on `IdentityChangedListener`) delivers a zeroed MAC on Aware, and
  `WIFI_P2P_THIS_DEVICE_CHANGED_ACTION` carries an anonymized MAC.
  Real MACs need `LOCAL_MAC_ADDRESS` (signature) or
  `ACCESS_FINE_LOCATION` explicitly.

Keep `ACCESS_FINE_LOCATION` with `android:maxSdkVersion="32"` for
backward compat: pre-33 discovery always needed precise location plus
Location Mode ON. Discovery calls still need Location Mode enabled on
some versions even with `NEARBY_WIFI_DEVICES`.

### Supporting permissions

| Permission | Protection | Use |
|---|---|---|
| `ACCESS_WIFI_STATE`, `CHANGE_WIFI_STATE` | normal | P2P `initialize()`, Aware attach/publish/subscribe |
| `CHANGE_NETWORK_STATE` | normal | `ConnectivityManager.requestNetwork` for Aware data paths |
| `INTERNET` | normal | Any Java socket, even off-Internet |
| `ACCESS_FINE_LOCATION` | dangerous | Pre-33 discovery. Real MACs. RTT to APs. `WifiManager` scan APIs |
| `ACCESS_LOCAL_NETWORK` | dangerous (API 37) | LAN sockets incl. mDNS/NsdManager on targetSdk 37+ |
| `LOCAL_MAC_ADDRESS` | signature | Real P2P device/GO MACs |
| `MANAGE_WIFI_NETWORK_SELECTION` | signature | Aware suspend/resume SystemApis |
| `SATELLITE_COMMUNICATION` | signature | SatelliteManager SystemApis |
| `READ_BASIC_PHONE_STATE` | normal (API 36) | Satellite state listener (or `READ_PHONE_STATE`/privileged/carrier) |
| `FOREGROUND_SERVICE` + `FOREGROUND_SERVICE_CONNECTED_DEVICE` | normal | Long-running P2P/Aware work. `connectedDevice` FGS type mandatory on targetSdk 34+ |

### ACCESS_LOCAL_NETWORK (API 37)

Starting Android 17, apps targeting API 37+ have local-network
traffic blocked by default: TCP/UDP unicast, multicast/broadcast,
mDNS, and `NsdManager`. Enforcement is in the networking stack, so
raw sockets to on-LAN peers fail without the grant. Cellular and VPN
are excluded. LAN DNS on :53 is exempt. Expect it to gate socket
traffic over Wi-Fi Direct group and Aware link interfaces for
targetSdk 37 apps, and request it alongside `NEARBY_WIFI_DEVICES`.

Android 16 already carries the enforcement behind a compat flag for
testing: `adb shell am compat enable RESTRICT_LOCAL_NETWORK <pkg>`
plus a reboot. During the opt-in phase the platform relies on
`NEARBY_WIFI_DEVICES` as a temporary stand-in. Denial count for
`ACCESS_LOCAL_NETWORK` resets with the NEARBY_DEVICES group, so a
user who denied twice can be re-prompted after the reset window.

## API-level matrix

| API | Android | What lands |
|---|---|---|
| 26 | 8.0 | `android.net.wifi.aware`: attach, publish/subscribe, messaging, open data paths |
| 29 | 10 | `WifiAwareNetworkSpecifier.Builder`, `WifiAwareNetworkInfo`, min/max-distance geofence |
| 30 | 11 | `Characteristics.getSupportedCipherSuites`. P2P `getNetworkId`, persistent groups |
| 31 | 12 | `getAvailableAwareResources`. OOB `createNetworkSpecifier*` deprecated |
| 33 | 13 | `NEARBY_WIFI_DEVICES`. `WifiAwareDataPathSecurityConfig`. Instant comm mode. Session-count characteristics |
| 34 | 14 | NAN pairing + bootstrapping (`AwarePairingConfig`, `initiatePairingRequest`). Mandatory FGS types |
| 35 | 15 | AOSP satellite stack (hidden `SatelliteManager`, `FEATURE_TELEPHONY_SATELLITE`). `TRANSPORT_SATELLITE`/`NET_CAPABILITY_NOT_BANDWIDTH_CONSTRAINED` (also via U Extensions 12). `WifiP2pListener` |
| 36 | 16 | `SatelliteManager` public + `SatelliteStateChangeListener` + `READ_BASIC_PHONE_STATE`. `AwarePairingConfig.setSupportedCipherSuites` (NCS_PK_PASN_*). P2P R2: pairing bootstrapping, PCC mode, USD discovery |
| 36.1 | 16 minor | `PROPERTY_SATELLITE_DATA_OPTIMIZED` manifest opt-in |
| 37 | 17 | `ACCESS_LOCAL_NETWORK` enforcement. Aware in-band data-path requests (`AwareDataPathRequest`). `NtnSignalStrength` public |

## Flow at a glance

- **Aware:** `WifiAwareManager.attach()` -> `publish`/`subscribe`
  session -> `sendMessage` for short payloads or
  `WifiAwareNetworkSpecifier` + `requestNetwork` for a socket-capable
  data path. [references/wifi-aware.md](references/wifi-aware.md)
- **Direct:** `WifiP2pManager.initialize()` -> `discoverPeers` ->
  `connect` -> `requestConnectionInfo` -> `groupOwnerAddress` ->
  plain sockets. [references/wifi-direct.md](references/wifi-direct.md)
- **Satellite:** `hasSystemFeature(FEATURE_TELEPHONY_SATELLITE)` ->
  `SatelliteManager.registerStateChangeListener` for modem state.
  Everything send-side is carrier-gated.
  [references/satellite.md](references/satellite.md)

## Common pitfalls

- No `NEARBY_WIFI_DEVICES` grant on API 33+ => `SecurityException` or
  `onFailure` on the first discovery call.
- Declaring `NEARBY_WIFI_DEVICES` without `neverForLocation` silently
  adds an `ACCESS_FINE_LOCATION` requirement.
- Aware `sendMessage` payloads are small (~255 bytes), unreliable, and
  unencrypted. Bulk or confidential data needs a secured data path.
- P2P allows one group at a time. Clients learn only the GO's
  address, not each other's.
- P2P `discoverPeers` needs Location Mode ON on older releases even
  with the new permission.
- Satellite feature detection is not enough to send. Without carrier
  privileges, the send APIs are unreachable.
