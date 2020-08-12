package api

// FirmwareMetadata represents a firmware image and related info.
type FirmwareMetadata struct {
	// Raw is the bytestream this object was parsed from.
	Raw []byte
	// Signature is the bytestream of the signature over the Raw field above.
	Signature []byte

	////// What's this firmware for? //////

	// TODO: possibly want a vendor/device/publisher model?

	// DeviceName is a human readable description of the target device.
	DeviceName string

	// DeviceID is a unique machine-readable ID for the target device.
	// Monitors should check that DeviceName and DeviceID correlate.
	DeviceID uint64

	////// Who published it //////

	// FirmwarePublisher is a human-readable desciption of the publisher.
	FirmwarePublisher string

	// FirmwarePublickeyHash is a hint for signing identity.
	// Monitors should check that this is indeed the key which signed
	// this metadata, *and* that the FirmwarePublisher string matches the key's
	// identity.
	FirmwarePublickeyHash []byte

	////// What's its identity? //////

	// FirmwareType identifies where it runs within the device.
	FirmwareType string

	// FirmwareRevision specifies which version of firmware this is.
	// TODO: string?
	FirmwareRevision uint64

	// FirmwareImageSHA512 is the SHA512 hash over the firmware image as it will
	// be delievered.
	FirmwareImageSHA512 []byte

	// ExpectedFirmwareMeasurement represents the expected measured state of the
	// device once the firmware is installed.
	ExpectedFirmwareMeasurement []byte

	///// What's its provenance? //////
	//
	// Given reproducible/hermetic builds, interested parties could "easily"/"cheaply" verify this claim.
	// Even without reproducible builds, it would be possible for interested parties to show that reversed sections of firmware could not have come from the claimed source.
	//
	// Probably need more info in here.

	// BuildFrom describes the source repo and commit from which this firmware was built.
	// e.g. "github.com/betterthanlife/talkie_firmware@4F232AE3B"
	// TODO: probably break this out into a structure.
	BuiltFrom string

	// BuildFlags is the set of "out of band" flags/env etc. which was provided during build time.
	// TODO: probably need to break this out into a structure.
	BuildFlags string

	// BuildTimestamp is the time at which this build was published in RFC3339 format.
	// e.g. "1985-04-12T23:20:50.52Z"
	BuildTimestamp string
}
