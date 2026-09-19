# Wi-Fi Aware (NAN) reference

Package `android.net.wifi.aware`, added API 26. NAN = Neighbor
Awareness Networking. Discovery runs in a shared cluster. The radio
disables Aware when the last attached app detaches.

## Entry point

`WifiAwareManager` via `getSystemService(Context.WIFI_AWARE_SERVICE)`.

- `attach(AttachCallback, Handler)` or `attach(callback,
  IdentityChangedListener, handler)` joins the cluster. Each call
  yields a distinct `WifiAwareSession`. All other calls wait for
  `AttachCallback.onAttached(WifiAwareSession)`.
- `IdentityChangedListener.onIdentityChanged(byte[] mac)` delivers
  the rotating discovery MAC. The real MAC only arrives if the app
  holds `ACCESS_FINE_LOCATION`. Otherwise all zeros.
- `getCharacteristics()`, `getAvailableAwareResources()` (API 31:
  free data paths/publish/subscribe counts),
  `isInstantCommunicationModeSupported()` (API 33).
- Gate: `hasSystemFeature(FEATURE_WIFI_AWARE)` plus runtime
  `isAvailable()`/`ACTION_WIFI_AWARE_STATE_CHANGED`.

## Discovery

`session.publish(PublishConfig, cb, handler)` ->
`PublishDiscoverySession`. `session.subscribe(SubscribeConfig, cb,
handler)` -> `SubscribeDiscoverySession`. Update with
`updatePublish`/`updateSubscribe`. Release with
`DiscoverySession.close()`.

`PublishConfig.Builder`: `setServiceName`, `setServiceSpecificInfo`,
`setMatchFilter`, `setPublishType` (`PUBLISH_TYPE_UNSOLICITED`
broadcasts by default. `PUBLISH_TYPE_SOLICITED` stays silent and
answers active subscribers), `setTtlSec`,
`setTerminateNotificationEnabled`, `setRangingEnabled` (API 29),
`setInstantCommunicationModeEnabled` (API 33),
`setDataPathSecurityConfig` (API 33: embeds a security-context ID in
publish frames, surfaced to subscribers as
`ServiceDiscoveryInfo.getScid()`), `setPairingConfig` (API 34).

`SubscribeConfig.Builder`: `setServiceName`,
`setServiceSpecificInfo`, `setMatchFilter`, `setSubscribeType`
(`PASSIVE`/`ACTIVE`), `setMinDistanceMm`/`setMaxDistanceMm` or
`setIngressDistanceMm`/`setEgressDistanceMm` (API 29 geofencing),
`setInstantCommunicationModeEnabled` (API 33), `setPairingConfig`
(API 34).

## Messaging

`DiscoverySession.sendMessage(PeerHandle, messageId, bytes)`.
`messageId` is app-chosen and echoed in `onMessageSendSucceeded`.
Payload capped by `Characteristics.getMaxServiceSpecificInfoLength()`
(~255 bytes). Delivery is unreliable: may drop, duplicate, or reorder.
Use it for handshakes, then move to a data path. Discovery traffic
(service name, info, match filter, messages) is **not encrypted**.

`PeerHandle` is an opaque privacy handle. `ParcelablePeer` persists a
peer reference.

## Data paths

```java
WifiAwareNetworkSpecifier spec =
    new WifiAwareNetworkSpecifier.Builder(discoverySession, peerHandle)
        .setPskPassphrase("shared-secret")
        .setPort(6000).setTransportProtocol(6) // TCP
        .build();
NetworkRequest req = new NetworkRequest.Builder()
        .addTransportType(NetworkCapabilities.TRANSPORT_WIFI_AWARE)
        .setNetworkSpecifier(spec).build();
connectivityManager.requestNetwork(req, callback);
```

Peer addressing arrives in `NetworkCapabilities.getTransportInfo()` as
`WifiAwareNetworkInfo` (API 29): `getPeerIpv6Addr()` (scoped
link-local IPv6), `getPort()`, `getTransportProtocol()`. Then open a
normal socket.

Security: default is open. `setPskPassphrase`, `setPmk`, or the
superset `setDataPathSecurityConfig` (API 33) pick cipher suites:
`NCS_SK_128/256` (shared key), `NCS_PK_128/256` (public key, needs
PMK+PMKID), `NONE`. Query `Characteristics.getSupportedCipherSuites()`
(API 30).

Roles are fixed: subscriber = INITIATOR, publisher = RESPONDER.
Deprecated helpers: `DiscoverySession.createNetworkSpecifierOpen/
Passphrase` (API 29) and OOB `WifiAwareSession.createNetworkSpecifier*`
(API 31).

**API 37 in-band requests:** `AwareDataPathRequest` plus
`PublishDiscoverySession.acceptDataPathRequest`/`rejectDataPathRequest`
and `SubscribeDiscoverySession.initiateDataPathRequest`, with
`onDataPathRequestReceived`/`onDataPathConnected`/
`onDataPathRequestFailed` callbacks. Removes the out-of-band
initiator-knowledge step.

## Pairing (Wi-Fi Aware R3/4.0, API 34)

`DiscoverySession.initiatePairingRequest(peerHandle, alias,
cipherSuite, password)` and `acceptPairingRequest(requestId, ...)`.
Null or empty password means opportunistic pairing. Callbacks:
`onPairingSetupRequestReceived`, `onPairingSetupSucceeded`,
`onPairingSetupFailed`.

`AwarePairingConfig` (API 34): `setBootstrappingMethods` +
`NCS_PK_PASN_128/256` cipher suites. Bootstrapping methods:
`OPPORTUNISTIC`, `PIN_CODE_DISPLAY`, `PASSPHRASE_DISPLAY`,
`QR_DISPLAY`, `QR_SCAN`, `PIN_CODE_KEYPAD`, `PASSPHRASE_KEYPAD`,
`NFC_TAG`, `NFC_READER`. `SERVICE_MANAGED` and `SKIPPED` added API 37.
`initiateBootstrappingRequest(peerHandle, method)` (API 34, `+byte[]`
overload API 37) runs Wi-Fi Aware 4.0 bootstrapping. Completion via
`onBootstrappingSucceeded`. Check
`Characteristics.isAwarePairingSupported()` and
`getSupportedPairingCipherSuites()` first.

## Ranging

Publisher opts in with `setRangingEnabled(true)`. Subscriber sets a
min/max or ingress/egress distance in mm. Geofence hits fire
`onServiceDiscoveredWithinRange`. Without publisher opt-in, discovery
proceeds un-ranged.

Direct ranging: `WifiRttManager.startRanging(RangingRequest, ...)`
(API 28), `RangingRequest.Builder.addWifiAwarePeer(PeerHandle)` or
`addWifiAwarePeer(MacAddress)`. Needs `FEATURE_WIFI_RTT`. Ranging
requests containing only Aware peers work with `NEARBY_WIFI_DEVICES`.
Ranging to APs still needs `ACCESS_FINE_LOCATION`.

## Limits and power

`Characteristics` exposes `getMaxServiceNameLength`,
`getMaxServiceSpecificInfoLength`, `getMaxMatchFilterLength`, and
`getNumberOfSupportedPublishSessions`/`SubscribeSessions`/`DataPaths`
(API 33). `getAvailableAwareResources()` (API 31) reports live
headroom. Power levers: publish/subscribe type, session TTL, and
instant communication mode (faster exchanges, more power).
`DiscoverySession.suspend()`/`resume()` exist but are `@SystemApi`
behind `MANAGE_WIFI_NETWORK_SELECTION` (signature), unusable by
third-party apps.
