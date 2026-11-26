// Package energyevse is the Energy EVSE cluster table of Matter 1.6.1: identity, feature map,
// attributes and commands with their conformance. The target maps no attribute of this cluster, so the table carries
// no transformed row, no encoder and no operation.
package energyevse

import "github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"

// Table returns a copy of the cluster table.
func Table() clusters.Table {
	return clusters.Table{
		ClusterID: 0x0099, Name: "Energy EVSE", Revision: 4,
		Features: []clusters.Feature{
			{Bit: 0, Code: "PREF", Name: "ChargingPreferences", Conformance: "M"},
			{Bit: 1, Code: "SOC", Name: "SoCReporting", Conformance: "O"},
			{Bit: 2, Code: "PNC", Name: "PlugAndCharge", Conformance: "O"},
			{Bit: 3, Code: "RFID", Name: "Rfid", Conformance: "O"},
			{Bit: 4, Code: "V2X", Name: "V2X", Conformance: "O"},
		},
		Attributes: []clusters.Attribute{
			{ID: 0x0000, Name: "State", MatterType: "StateEnum", Nullable: true, Conformance: "M", Access: "R V", Quality: "X", Mandatory: true},
			{ID: 0x0001, Name: "SupplyState", MatterType: "SupplyStateEnum", Nullable: false, Conformance: "M", Access: "R V", Quality: "", Mandatory: true},
			{ID: 0x0002, Name: "FaultState", MatterType: "FaultStateEnum", Nullable: false, Conformance: "M", Access: "R V", Quality: "", Mandatory: true},
			{ID: 0x0003, Name: "ChargingEnabledUntil", MatterType: "epoch-s", Nullable: true, Conformance: "M", Access: "R V", Quality: "X N", Mandatory: true},
			{ID: 0x0004, Name: "DischargingEnabledUntil", MatterType: "epoch-s", Nullable: true, Conformance: "V2X", Access: "R V", Quality: "X N", Mandatory: false},
			{ID: 0x0005, Name: "CircuitCapacity", MatterType: "amperage-mA", Nullable: false, Conformance: "M", Access: "R V", Quality: "N", Mandatory: true},
			{ID: 0x0006, Name: "MinimumChargeCurrent", MatterType: "amperage-mA", Nullable: false, Conformance: "M", Access: "R V", Quality: "N", Mandatory: true},
			{ID: 0x0007, Name: "MaximumChargeCurrent", MatterType: "amperage-mA", Nullable: false, Conformance: "M", Access: "R V", Quality: "N", Mandatory: true},
			{ID: 0x0008, Name: "MaximumDischargeCurrent", MatterType: "amperage-mA", Nullable: false, Conformance: "V2X", Access: "R V", Quality: "N", Mandatory: false},
			{ID: 0x0009, Name: "UserMaximumChargeCurrent", MatterType: "amperage-mA", Nullable: false, Conformance: "O", Access: "RW VM", Quality: "N", Mandatory: false},
			{ID: 0x000A, Name: "RandomizationDelayWindow", MatterType: "elapsed-s", Nullable: false, Conformance: "O", Access: "RW VM", Quality: "N", Mandatory: false},
			{ID: 0x0023, Name: "NextChargeStartTime", MatterType: "epoch-s", Nullable: true, Conformance: "PREF", Access: "R V", Quality: "X", Mandatory: false},
			{ID: 0x0024, Name: "NextChargeTargetTime", MatterType: "epoch-s", Nullable: true, Conformance: "PREF", Access: "R V", Quality: "X", Mandatory: false},
			{ID: 0x0025, Name: "NextChargeRequiredEnergy", MatterType: "energy-mWh", Nullable: true, Conformance: "PREF", Access: "R V", Quality: "X", Mandatory: false},
			{ID: 0x0026, Name: "NextChargeTargetSoC", MatterType: "percent", Nullable: true, Conformance: "PREF", Access: "R V", Quality: "X", Mandatory: false},
			{ID: 0x0027, Name: "ApproximateEvEfficiency", MatterType: "uint16", Nullable: true, Conformance: "[PREF]", Access: "RW VM", Quality: "X N", Mandatory: false},
			{ID: 0x0030, Name: "StateOfCharge", MatterType: "percent", Nullable: true, Conformance: "SOC", Access: "R V", Quality: "X", Mandatory: false},
			{ID: 0x0031, Name: "BatteryCapacity", MatterType: "energy-mWh", Nullable: true, Conformance: "SOC", Access: "R V", Quality: "X", Mandatory: false},
			{ID: 0x0032, Name: "VehicleId", MatterType: "string", Nullable: true, Conformance: "PNC", Access: "R V", Quality: "X", Mandatory: false},
			{ID: 0x0040, Name: "SessionId", MatterType: "uint32", Nullable: true, Conformance: "M", Access: "R V", Quality: "X N", Mandatory: true},
			{ID: 0x0041, Name: "SessionDuration", MatterType: "elapsed-s", Nullable: true, Conformance: "M", Access: "R V", Quality: "X N Q", Mandatory: true},
			{ID: 0x0042, Name: "SessionEnergyCharged", MatterType: "energy-mWh", Nullable: true, Conformance: "M", Access: "R V", Quality: "X N Q", Mandatory: true},
			{ID: 0x0043, Name: "SessionEnergyDischarged", MatterType: "energy-mWh", Nullable: true, Conformance: "V2X", Access: "R V", Quality: "X N Q", Mandatory: false},
			{ID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Conformance: "M", Access: "R V", Mandatory: true},
			{ID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Conformance: "M", Access: "R V", Mandatory: true},
		},
		Commands: []clusters.Command{
			{ID: 0x00, Name: "GetTargetsResponse", Direction: "response", Conformance: "PREF"},
			{ID: 0x01, Name: "Disable", Direction: "request → status", Conformance: "M"},
			{ID: 0x02, Name: "EnableCharging", Direction: "request → status", Conformance: "M"},
			{ID: 0x03, Name: "EnableDischarging", Direction: "request → status", Conformance: "V2X"},
			{ID: 0x04, Name: "StartDiagnostics", Direction: "request → status", Conformance: "O"},
			{ID: 0x05, Name: "SetTargets", Direction: "request → status", Conformance: "PREF"},
			{ID: 0x06, Name: "GetTargets", Direction: "request → GetTargetsResponse", Conformance: "PREF"},
			{ID: 0x07, Name: "ClearTargets", Direction: "request → status", Conformance: "PREF"},
		},
	}
}
