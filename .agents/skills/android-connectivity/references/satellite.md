# Satellite / NTN reference

Package `android.telephony.satellite`. The AOSP NTN stack landed in
Android 15 (API 35): framework state machine, routing overrides,
`ISatellite` modem HAL (AIDL), dynamic carrier config via
`CarrierConfigManager` XML or GSMA TS.43 entitlement servers.

Feature detection: `hasSystemFeature(FEATURE_TELEPHONY_SATELLITE)`
plus `getSystemService(Context.SATELLITE_SERVICE)`.

## What is public

`SatelliteManager` was hidden/`@SystemApi` in API 35 and became a
public `@SystemService` in **API 36** under the
`satellite_state_change_listener` flag.

Third-party apps can use:

- **`registerStateChangeListener(Executor,
  SatelliteStateChangeListener)` / `unregisterStateChangeListener`**
  (API 36). `onEnabledStateChanged(boolean)` reports satellite modem
  enabled state at the device level, not per-SIM. Requires any of:
  `READ_BASIC_PHONE_STATE` (new normal permission, API 36),
  `READ_PRIVILEGED_PHONE_STATE`, `READ_PHONE_STATE`, or carrier
  privileges.
- **`PROPERTY_SATELLITE_DATA_OPTIMIZED`** (API 36.1): manifest
  meta-data `android.telephony.PROPERTY_SATELLITE_DATA_OPTIMIZED`
  naming the package opts the app into constrained satellite
  networks and lists it among satellite-enabled apps in Settings.
- **Network detection (API 36+):**
  `NetworkCapabilities.TRANSPORT_SATELLITE` and the inverse
  `NET_CAPABILITY_NOT_BANDWIDTH_CONSTRAINED` bit let an app detect a
  low-bandwidth satellite link through `ConnectivityManager` and
  degrade gracefully.
- **`NtnSignalStrength`** (public API 37): `NONE/POOR/MODERATE/GOOD/
  GREAT`, `getLevel()`.
- **`TelephonyCallback.CarrierRoamingNtnListener`** (public interface,
  API 37): `onCarrierRoamingNtnModeChanged`,
  `onCarrierRoamingNtnSignalStrengthChanged`, plus eligibility and
  available-services callbacks. Caveat: the underlying
  `EVENT_CARRIER_ROAMING_NTN_*` constants remain `@SystemApi` +
  `READ_PHONE_STATE` + `FLAG_SATELLITE_SYSTEM_APIS` in the
  android17-release tree, so whether a third-party app can actually
  register and receive these events is unverified and likely
  privileged in practice.

## What is carrier/system gated

Nearly everything else in `SatelliteManager` is `@SystemApi` +
`@RequiresPermission(SATELLITE_COMMUNICATION)` (signature) +
aconfig-flagged (`FLAG_OEM_ENABLED_SATELLITE_FLAG`,
`FLAG_CARRIER_ENABLED_SATELLITE_FLAG`):

- `requestIsSupported`, `requestIsProvisioned`,
  `requestEnabled(EnableRequestAttributes)` with `isDemoMode`/
  `isEmergencyMode`, `requestIsEnabled`, `requestNtnSignalStrength`
- datagram send/receive (`SatelliteDatagramCallback`)
- modem-state callbacks (`SatelliteModemStateCallback`: OFF/IDLE/
  LISTENING/DATAGRAM_TRANSFERRING/NOT_CONNECTED/CONNECTED)
- pointing-info start/stop, provisioning
  (`provisionSatelliteService`/`deprovisionSatelliteService`),
  attach-restriction controls, next-visibility timing

Only carrier-privileged or system apps can call these. Third-party
apps **cannot send arbitrary satellite messages or data, point at
the sky, or query visibility**.

## Device and carrier reality

- Devices: Pixel 9 and later on Android 15+, select Samsung Galaxy
  (S24/S25-era).
- Services: Pixel Satellite SOS is an emergency-mode flow to
  emergency providers. Carrier satellite SMS/RCS rides Skylo
  (Verizon/AT&T via Google Messages) or Starlink direct-to-cell
  (T-Mobile "T-Satellite"). T-Mobile's constrained data tier for
  whitelisted optimized apps launched Oct 2025, which is what
  `PROPERTY_SATELLITE_DATA_OPTIMIZED` targets.
- Support tiers in the platform: SMS/MMS/RCS over NTN, constrained
  "light" data for allowlisted apps, and unconstrained IP data where
  carrier and hardware allow.

## Practical posture for app code

- Detect `FEATURE_TELEPHONY_SATELLITE` before calling anything in the
  package.
- Register the state listener to know when the modem has satellite
  enabled, but do not assume that means your sockets will route.
- Check `NetworkCapabilities` for `TRANSPORT_SATELLITE` or missing
  `NOT_BANDWIDTH_CONSTRAINED` before bulk transfers.
- If the product needs satellite messaging, it needs a carrier or
  OEM partnership. There is no public send path.
