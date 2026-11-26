// Package deviceenergymanagement is the Device Energy Management cluster table of Matter 1.6.1: identity, feature map,
// attributes and commands with their conformance, and the transformed rows the target maps.
package deviceenergymanagement

import "github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"

// Table returns a copy of the cluster table.
func Table() clusters.Table {
	return clusters.Table{
		ClusterID: 0x0098, Name: "Device Energy Management", Revision: 4,
		Features: []clusters.Feature{
			{Bit: 0, Code: "PA", Name: "PowerAdjustment", Conformance: "O"},
			{Bit: 1, Code: "PFR", Name: "PowerForecastReporting", Conformance: "[!PA].a, STA | PAU | FA | CON, O"},
			{Bit: 2, Code: "SFR", Name: "StateForecastReporting", Conformance: "[!PA].a"},
			{Bit: 3, Code: "STA", Name: "StartTimeAdjustment", Conformance: "O"},
			{Bit: 4, Code: "PAU", Name: "Pausable", Conformance: "O"},
			{Bit: 5, Code: "FA", Name: "ForecastAdjustment", Conformance: "O"},
			{Bit: 6, Code: "CON", Name: "ConstraintBasedAdjustment", Conformance: "O"},
		},
		Attributes: []clusters.Attribute{
			{ID: 0x0000, Name: "EsaType", MatterType: "ESATypeEnum", Nullable: false, Conformance: "M", Access: "R V", Quality: "F", Mandatory: true},
			{ID: 0x0001, Name: "EsaCanGenerate", MatterType: "bool", Nullable: false, Conformance: "M", Access: "R V", Quality: "F", Mandatory: true},
			{ID: 0x0002, Name: "EsaState", MatterType: "ESAStateEnum", Nullable: false, Conformance: "M", Access: "R V", Quality: "", Mandatory: true},
			{ID: 0x0003, Name: "AbsMinPower", MatterType: "power-mW", Nullable: false, Conformance: "M", Access: "R V", Quality: "", Mandatory: true},
			{ID: 0x0004, Name: "AbsMaxPower", MatterType: "power-mW", Nullable: false, Conformance: "M", Access: "R V", Quality: "", Mandatory: true},
			{ID: 0x0005, Name: "PowerAdjustmentCapability", MatterType: "PowerAdjustCapabilityStruct", Nullable: true, Conformance: "PA", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x0006, Name: "Forecast", MatterType: "ForecastStruct", Nullable: true, Conformance: "PFR | SFR", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x0007, Name: "OptOutState", MatterType: "OptOutStateEnum", Nullable: false, Conformance: "PA | STA | PAU | FA | CON", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Conformance: "M", Access: "R V", Mandatory: true},
			{ID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Conformance: "M", Access: "R V", Mandatory: true},
		},
		Commands: []clusters.Command{
			{ID: 0x00, Name: "PowerAdjustRequest", Direction: "request → status", Conformance: "PA"},
			{ID: 0x01, Name: "CancelPowerAdjustRequest", Direction: "request → status", Conformance: "PA"},
			{ID: 0x02, Name: "StartTimeAdjustRequest", Direction: "request → status", Conformance: "STA"},
			{ID: 0x03, Name: "PauseRequest", Direction: "request → status", Conformance: "PAU"},
			{ID: 0x04, Name: "ResumeRequest", Direction: "request → status", Conformance: "PAU"},
			{ID: 0x05, Name: "ModifyForecastRequest", Direction: "request → status", Conformance: "FA"},
			{ID: 0x06, Name: "RequestConstraintBasedForecast", Direction: "request → status", Conformance: "CON"},
			{ID: 0x07, Name: "CancelRequest", Direction: "request → status", Conformance: "STA | FA | CON"},
		},
	}
}
