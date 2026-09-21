# Wi-Fi Direct (P2P) reference

Package `android.net.wifi.p2p`, added API 14. One device becomes group
owner (GO), runs DHCP, and hands out addresses. Clients connect to the
GO's address. Only one group can be active at a time.

## Core flow

```java
WifiP2pManager mgr = (WifiP2pManager) getSystemService(Context.WIFI_P2P_SERVICE);
WifiP2pManager.Channel channel = mgr.initialize(this, getMainLooper(), null);
mgr.discoverPeers(channel, actionListener);
// WIFI_P2P_PEERS_CHANGED_ACTION -> requestPeers -> WifiP2pDeviceList
mgr.connect(channel, config, actionListener);
// WIFI_P2P_CONNECTION_CHANGED_ACTION -> requestConnectionInfo
// -> WifiP2pInfo{groupFormed, groupOwnerAddress, isGroupOwner}
// -> plain Java sockets to groupOwnerAddress
```

- `initialize` needs `ACCESS_WIFI_STATE` + `CHANGE_WIFI_STATE` and is
  restricted inside the SDK Runtime sandbox since API 34.
- `WifiP2pInfo.groupOwnerAddress` is the GO's `InetAddress`
  (commonly `192.168.49.1`). Clients open sockets to it. The GO
  accepts. If the local device is GO, the address is anonymized
  unless the caller holds `LOCAL_MAC_ADDRESS` (signature).
- `createGroup` makes an autonomous GO (legacy-device hotspot).`requestGroupInfo` returns `WifiP2pGroup` (`getInterface`,
  `getNetworkName`, `getPassphrase`, `getFrequency`, `getClientList`,
  `getNetworkId` API 30, `getSecurityType` + `SECURITY_TYPE_WPA2_PSK`/
  `WPA3_SAE`/`WPA3_COMPATIBILITY` + `getGroupOwnerBssid` API 36). R2
  groups randomize interface addresses per connection.

## WifiP2pConfig

Legacy fields: `deviceAddress`, `wps` (`WpsInfo` `PBC`/`KEYPAD`/
`DISPLAY`), `groupOwnerIntent` (0-15, `GROUP_OWNER_INTENT_AUTO = -1`,
still the GO-inclination knob).

`WifiP2pConfig.Builder` (API 29): `setDeviceAddress(MacAddress)`,
`setNetworkName("DIRECT-xy...")` + `setPassphrase` (join or invite a
known group), `setGroupOperatingBand`/`setGroupOperatingFrequency`
(2.4/5/6 GHz or AUTO), `enablePersistentMode(boolean)` ->
`NETWORK_ID_PERSISTENT`/`TEMPORARY`, `setNetworkId(int)` for a
remembered group.

## Service discovery

Pre-connection local service discovery:
`addLocalService(Channel, WifiP2pServiceInfo, ...)` with
`WifiP2pDnsSdServiceInfo`/`WifiP2pDnsSdTxtRecord`/
`WifiP2pUpnpServiceInfo`, then `discoverServices`,
`setDnsSdResponseListeners`, `setServiceResponseListener`,
`setUpnpServiceResponseListener`.

## Broadcasts

All require `ACCESS_WIFI_STATE` + (`ACCESS_FINE_LOCATION` or
`NEARBY_WIFI_DEVICES`):

`WIFI_P2P_STATE_CHANGED_ACTION`, `WIFI_P2P_DISCOVERY_CHANGED_ACTION`,
`WIFI_P2P_PEERS_CHANGED_ACTION`, `WIFI_P2P_CONNECTION_CHANGED_ACTION`
(extras `EXTRA_WIFI_P2P_INFO`, `EXTRA_NETWORK_INFO`,
`EXTRA_WIFI_P2P_GROUP`), `WIFI_P2P_THIS_DEVICE_CHANGED_ACTION`
(`EXTRA_WIFI_P2P_DEVICE` carries an anonymized MAC. Real MAC needs
`LOCAL_MAC_ADDRESS` + `requestDeviceInfo`, API 29).

## Newer APIs (R2 / API 35-37)

- **API 35:** `WifiP2pListener` consolidates state callbacks. `requestP2pState`, `requestDiscoveryState`, `getListenState`.
- **API 36:** `WifiP2pPairingBootstrappingConfig` +
  `WifiP2pConfig.Builder.setPairingBootstrappingConfig` replace the
  legacy WPS dialog with programmatic pairing:
  `DISPLAY_PASSPHRASE`, `DISPLAY_PINCODE`, `KEYPAD_PASSPHRASE`,
  `KEYPAD_PINCODE`, `OPPORTUNISTIC`, `OUT_OF_BAND` (e.g. password over
  BLE). `setPairingDiscoveryChannelFrequencyMhz` for OOB. `setAuthorizeConnectionFromPeerEnabled` to pre-authorize an
  incoming request.
- **API 36 PCC mode:** `isPccModeSupported()`,
  `setPccModeConnectionType` (`LEGACY_ONLY`/`LEGACY_OR_R2`/`R2_ONLY`, R2 groups are WPA3-Personal), `isWiFiDirectR2Supported()`.
- **API 36 USD service discovery:** `discoverUsdBasedServices`,
  `startUsdBasedLocalServiceAdvertisement`, plus config classes.
- `setGroupClientIpProvisioningMode` for client IP provisioning
  incl. IPv6 link-local. `discoverPeersOnSocialChannels`/
  `discoverPeersOnSpecificFrequency`, `WifiP2pDiscoveryConfig`,
  `WifiP2pDirInfo`, `WifiP2pWfdInfo` (API 36-37).

## Caveats

- One group at a time. The framework reuses or tears down an existing
  group rather than running concurrent groups. No multi-hop. Clients
  see only the GO.
- `discoverPeers`, `discoverServices`, `requestPeers` need Location
  Mode ON on older releases, not just permissions.
- Legacy `connect()` can surface the system pairing UI. R2
  bootstrapping (API 36) gives an app-controlled pairing flow.
- No `NEARBY_WIFI_DEVICES` grant on API 33+ => `SecurityException` or
  `onFailure`.
- Clients only learn the GO's address. Peer-to-peer client routing
  goes through the GO.
