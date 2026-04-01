package config

// UserConfigFileName is the global settings file inside GlobalConfigDir (SPEC §1.3).
const UserConfigFileName = "config.json"

// LocalConfigFileName is the gitignored local overlay in project root (SPEC §1.2).
const LocalConfigFileName = ".rabbit-code.local.json"

// PluginBaseConfigFileName is the lowest-priority settings layer (P2.1.6), under GlobalConfigDir.
const PluginBaseConfigFileName = "plugin-settings.base.json"

// EnvFlagJSON is RABBIT_CODE_SETTINGS_JSON: merged as flagSettings layer (highest except policy).
const EnvFlagJSON = "RABBIT_CODE_SETTINGS_JSON"

// EnvPolicyMDMJSON / EnvPolicyRemoteJSON are optional enterprise policy layers (P2.3.1).
// Merge order (later wins): HKCU stub → managed files → MDM env → remote env.
const EnvPolicyMDMJSON = "RABBIT_CODE_POLICY_MDM_JSON"
const EnvPolicyRemoteJSON = "RABBIT_CODE_POLICY_REMOTE_JSON"

const managedSettingsFile = "managed-settings.json"
const managedSettingsDropInDir = "managed-settings.d"
