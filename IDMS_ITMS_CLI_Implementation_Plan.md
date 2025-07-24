# IDMS/ITMS Support Implementation Plan for ROSA CLI

## Overview

This document outlines the implementation plan for adding `ImageDigestMirrorSet` (IDMS) and `ImageTagMirrorSet` (ITMS) support to the ROSA CLI using the `feat/IDMS-ITMS-support` branch. This feature will provide modern image mirroring capabilities to replace the deprecated `ImageContentSourcePolicy` (ICSP) starting with OpenShift 4.13+.

## Background

The backend `uhc-clusters-service` already has complete IDMS/ITMS support implemented. This plan focuses on adding corresponding CLI functionality to the ROSA client to expose these capabilities to users.

### Key Benefits
- **Modern Image Mirroring**: Replace deprecated ICSP with IDMS/ITMS
- **Granular Control**: Digest-specific and tag-specific mirroring
- **Platform Safety**: Protection against platform registry misconfiguration
- **Backward Compatibility**: Works alongside existing `RegistrySources`

## Implementation Phases

## Phase 1: Core Data Model Extensions

### 1.1 Update OCM Spec Structure
**File**: `pkg/ocm/clusters.go`

Add new IDMS/ITMS fields to the `Spec` struct:

```go
// Registry Config (existing fields)
AllowedRegistries          []string
BlockedRegistries          []string
InsecureRegistries         []string
AllowedRegistriesForImport string
PlatformAllowlist          string
AdditionalTrustedCaFile    string
AdditionalTrustedCa        map[string]string

// NEW: IDMS/ITMS Support
ImageDigestMirrorSets      []ImageDigestMirrorSet
ImageTagMirrorSets         []ImageTagMirrorSet
```

Add new type definitions:
```go
type ImageDigestMirrorSet struct {
    Name    string        `json:"name"`
    Mirrors []ImageMirror `json:"mirrors"`
}

type ImageTagMirrorSet struct {
    Name    string        `json:"name"`
    Mirrors []ImageMirror `json:"mirrors"`
}

type ImageMirror struct {
    Source           string   `json:"source"`
    MirrorsByDigest  []string `json:"mirrors_by_digest,omitempty"`
    MirrorsByTag     []string `json:"mirrors_by_tag,omitempty"`
}
```

### 1.2 Update Registry Config Builder
**File**: `pkg/ocm/registry_config.go`

Extend `BuildRegistryConfig()` function to handle IDMS/ITMS:

```go
func BuildRegistryConfig(spec Spec) (*cmv1.ClusterRegistryConfigBuilder, error) {
    // Existing logic for legacy registry sources...
    
    // NEW: Add IDMS/ITMS support
    if len(spec.ImageDigestMirrorSets) > 0 || len(spec.ImageTagMirrorSets) > 0 {
        // Build IDMS configurations
        if len(spec.ImageDigestMirrorSets) > 0 {
            idmsList := make([]*cmv1.ImageDigestMirrorSetBuilder, 0, len(spec.ImageDigestMirrorSets))
            for _, idms := range spec.ImageDigestMirrorSets {
                // Convert to OCM SDK format
                idmsBuilder := cmv1.NewImageDigestMirrorSet().Name(idms.Name)
                
                // Add mirrors
                mirrorBuilders := make([]*cmv1.ImageMirrorBuilder, 0, len(idms.Mirrors))
                for _, mirror := range idms.Mirrors {
                    mirrorBuilder := cmv1.NewImageMirror().
                        Source(mirror.Source).
                        MirrorsByDigest(mirror.MirrorsByDigest...)
                    mirrorBuilders = append(mirrorBuilders, mirrorBuilder)
                }
                idmsBuilder.Mirrors(mirrorBuilders...)
                idmsList = append(idmsList, idmsBuilder)
            }
            clusterRegistryConfig.ImageDigestMirrorSets(idmsList...)
        }
        
        // Build ITMS configurations  
        if len(spec.ImageTagMirrorSets) > 0 {
            itmsList := make([]*cmv1.ImageTagMirrorSetBuilder, 0, len(spec.ImageTagMirrorSets))
            for _, itms := range spec.ImageTagMirrorSets {
                // Convert to OCM SDK format
                itmsBuilder := cmv1.NewImageTagMirrorSet().Name(itms.Name)
                
                // Add mirrors
                mirrorBuilders := make([]*cmv1.ImageMirrorBuilder, 0, len(itms.Mirrors))
                for _, mirror := range itms.Mirrors {
                    mirrorBuilder := cmv1.NewImageMirror().
                        Source(mirror.Source).
                        MirrorsByTag(mirror.MirrorsByTag...)
                    mirrorBuilders = append(mirrorBuilders, mirrorBuilder)
                }
                itmsBuilder.Mirrors(mirrorBuilders...)
                itmsList = append(itmsList, itmsBuilder)
            }
            clusterRegistryConfig.ImageTagMirrorSets(itmsList...)
        }
    }
    
    return clusterRegistryConfig, nil
}
```

## Phase 2: CLI Flag Extensions

### 2.1 Add New CLI Flags
**File**: `pkg/clusterregistryconfig/flags.go`

Add new flag constants and struct fields:
```go
const (
    // Existing flags...
    allowedRegistriesFlag          = "registry-config-allowed-registries"
    insecureRegistriesFlag         = "registry-config-insecure-registries"
    blockedRegistriesFlag          = "registry-config-blocked-registries"
    platformAllowlistFlag          = "registry-config-platform-allowlist"
    additionalTrustedCaPathFlag    = "registry-config-additional-trusted-ca"
    allowedRegistriesForImportFlag = "registry-config-allowed-registries-for-import"
    
    // NEW: IDMS/ITMS flags
    imageDigestMirrorSetsFlag = "registry-config-image-digest-mirror-sets"
    imageTagMirrorSetsFlag    = "registry-config-image-tag-mirror-sets"
)

type ClusterRegistryConfigArgs struct {
    // Existing fields...
    allowedRegistries          []string
    blockedRegistries          []string
    insecureRegistries         []string
    allowedRegistriesForImport string
    platformAllowlist          string
    additionalTrustedCa        string
    
    // NEW: IDMS/ITMS fields
    imageDigestMirrorSets []string
    imageTagMirrorSets    []string
}
```

Add flag definitions to `AddClusterRegistryConfigFlags()`:
```go
cmd.Flags().StringSliceVar(
    &args.imageDigestMirrorSets,
    imageDigestMirrorSetsFlag,
    nil,
    "ImageDigestMirrorSet configurations for digest-based image mirroring. "+
        "Format: 'name:source=mirror1,mirror2'. Multiple sets can be specified. "+
        "Example: 'company-mirrors:registry.company.com/openshift=mirror1.company.com/openshift,mirror2.company.com/openshift'",
)

cmd.Flags().StringSliceVar(
    &args.imageTagMirrorSets,
    imageTagMirrorSetsFlag,
    nil,
    "ImageTagMirrorSet configurations for tag-based image mirroring. "+
        "Format: 'name:source=mirror1,mirror2'. Multiple sets can be specified. "+
        "Example: 'tag-mirrors:docker.io/library=mirror.company.com/docker-library'",
)
```

### 2.2 Add Interactive Mode Support
Update `GetClusterRegistryConfigOptions()` to handle interactive input for IDMS/ITMS:

```go
if enableRegistriesConfig && interactive.Enabled() {
    // Existing interactive prompts for legacy registry config...
    
    // NEW: IDMS interactive prompts
    result.imageDigestMirrorSets, err = interactive.GetStringSlice(interactive.Input{
        Question: "Image Digest Mirror Sets",
        Help:     cmd.Lookup(imageDigestMirrorSetsFlag).Usage,
        Default:  defaultImageDigestMirrorSets,
        Validators: []interactive.Validator{
            validateImageMirrorSetFormat,
        },
    })
    if err != nil {
        return nil, fmt.Errorf("Expected valid IDMS configuration: %s", err)
    }
    
    // NEW: ITMS interactive prompts
    result.imageTagMirrorSets, err = interactive.GetStringSlice(interactive.Input{
        Question: "Image Tag Mirror Sets",
        Help:     cmd.Lookup(imageTagMirrorSetsFlag).Usage,
        Default:  defaultImageTagMirrorSets,
        Validators: []interactive.Validator{
            validateImageMirrorSetFormat,
        },
    })
    if err != nil {
        return nil, fmt.Errorf("Expected valid ITMS configuration: %s", err)
    }
}
```

### 2.3 Add Parser Functions
Add functions to parse IDMS/ITMS string formats into structured data:

```go
func parseImageDigestMirrorSets(input []string) ([]ocm.ImageDigestMirrorSet, error) {
    var result []ocm.ImageDigestMirrorSet
    
    for _, entry := range input {
        if entry == "" {
            continue
        }
        
        // Parse format: "name:source=mirror1,mirror2"
        parts := strings.SplitN(entry, ":", 2)
        if len(parts) != 2 {
            return nil, fmt.Errorf("invalid IDMS format: %s (expected 'name:source=mirror1,mirror2')", entry)
        }
        
        name := parts[0]
        sourceMirrors := strings.SplitN(parts[1], "=", 2)
        if len(sourceMirrors) != 2 {
            return nil, fmt.Errorf("invalid source=mirrors format: %s", parts[1])
        }
        
        source := sourceMirrors[0]
        mirrors := strings.Split(sourceMirrors[1], ",")
        
        // Validate mirrors
        for i, mirror := range mirrors {
            mirrors[i] = strings.TrimSpace(mirror)
            if mirrors[i] == "" {
                return nil, fmt.Errorf("empty mirror specified for source %s", source)
            }
        }
        
        result = append(result, ocm.ImageDigestMirrorSet{
            Name: name,
            Mirrors: []ocm.ImageMirror{
                {
                    Source:          source,
                    MirrorsByDigest: mirrors,
                },
            },
        })
    }
    
    return result, nil
}

func parseImageTagMirrorSets(input []string) ([]ocm.ImageTagMirrorSet, error) {
    var result []ocm.ImageTagMirrorSet
    
    for _, entry := range input {
        if entry == "" {
            continue
        }
        
        // Parse format: "name:source=mirror1,mirror2"
        parts := strings.SplitN(entry, ":", 2)
        if len(parts) != 2 {
            return nil, fmt.Errorf("invalid ITMS format: %s (expected 'name:source=mirror1,mirror2')", entry)
        }
        
        name := parts[0]
        sourceMirrors := strings.SplitN(parts[1], "=", 2)
        if len(sourceMirrors) != 2 {
            return nil, fmt.Errorf("invalid source=mirrors format: %s", parts[1])
        }
        
        source := sourceMirrors[0]
        mirrors := strings.Split(sourceMirrors[1], ",")
        
        // Validate mirrors
        for i, mirror := range mirrors {
            mirrors[i] = strings.TrimSpace(mirror)
            if mirrors[i] == "" {
                return nil, fmt.Errorf("empty mirror specified for source %s", source)
            }
        }
        
        result = append(result, ocm.ImageTagMirrorSet{
            Name: name,
            Mirrors: []ocm.ImageMirror{
                {
                    Source:        source,
                    MirrorsByTag:  mirrors,
                },
            },
        })
    }
    
    return result, nil
}
```

### 2.4 Update Flag Processing
Add new getter function in `flags.go`:

```go
func GetImageMirrorSetArgs(args *ClusterRegistryConfigArgs) ([]ocm.ImageDigestMirrorSet, []ocm.ImageTagMirrorSet, error) {
    idmsList, err := parseImageDigestMirrorSets(args.imageDigestMirrorSets)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to parse IDMS: %v", err)
    }
    
    itmsList, err := parseImageTagMirrorSets(args.imageTagMirrorSets)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to parse ITMS: %v", err)
    }
    
    return idmsList, itmsList, nil
}
```

Update `IsClusterRegistryConfigSetViaCLI()` to include new flags:
```go
func IsClusterRegistryConfigSetViaCLI(cmd *pflag.FlagSet) bool {
    for _, parameter := range []string{
        allowedRegistriesFlag,
        insecureRegistriesFlag, 
        blockedRegistriesFlag, 
        platformAllowlistFlag,
        allowedRegistriesForImportFlag, 
        additionalTrustedCaPathFlag,
        imageDigestMirrorSetsFlag,    // NEW
        imageTagMirrorSetsFlag,       // NEW
    } {
        if cmd.Changed(parameter) {
            return true
        }
    }
    return false
}
```

## Phase 3: Command Integration

### 3.1 Update Cluster Creation Command
**File**: `cmd/create/cluster/cmd.go`

Update the cluster creation flow to handle IDMS/ITMS around line 3430:

```go
// Existing registry config processing
clusterRegistryConfigArgs, err = clusterregistryconfig.GetClusterRegistryConfigOptions(
    cmd.Flags(), clusterRegistryConfigArgs, isHostedCP, nil)
if err != nil {
    r.Reporter.Errorf("%s", err)
    os.Exit(1)
}

if clusterRegistryConfigArgs != nil {
    allowedRegistries, blockedRegistries, insecureRegistries,
        additionalTrustedCa, allowedRegistriesForImport,
        platformAllowlist := clusterregistryconfig.GetClusterRegistryConfigArgs(
        clusterRegistryConfigArgs)
        
    // NEW: Process IDMS/ITMS
    imageDigestMirrorSets, imageTagMirrorSets, err := 
        clusterregistryconfig.GetImageMirrorSetArgs(clusterRegistryConfigArgs)
    if err != nil {
        r.Reporter.Errorf("Failed to parse image mirror sets: %s", err)
        os.Exit(1)
    }
    
    // Set in cluster config
    clusterConfig.AllowedRegistries = allowedRegistries
    clusterConfig.BlockedRegistries = blockedRegistries
    clusterConfig.InsecureRegistries = insecureRegistries
    clusterConfig.PlatformAllowlist = platformAllowlist
    
    // NEW: Set IDMS/ITMS
    clusterConfig.ImageDigestMirrorSets = imageDigestMirrorSets
    clusterConfig.ImageTagMirrorSets = imageTagMirrorSets
    
    // Existing additional trusted CA logic...
    if additionalTrustedCa != "" {
        ca, err := clusterregistryconfig.BuildAdditionalTrustedCAFromInputFile(additionalTrustedCa)
        if err != nil {
            r.Reporter.Errorf("Failed to build the additional trusted ca from file %s, got error: %s", additionalTrustedCa, err)
            os.Exit(1)
        }
        clusterConfig.AdditionalTrustedCa = ca
        clusterConfig.AdditionalTrustedCaFile = additionalTrustedCa
    }
    clusterConfig.AllowedRegistriesForImport = allowedRegistriesForImport
}
```

### 3.2 Update Cluster Edit Command
**File**: `cmd/edit/cluster/cmd.go`

Similar updates for the edit command around line 690:

```go
clusterRegistryConfigArgs, err = clusterregistryconfig.GetClusterRegistryConfigOptions(
    cmd.Flags(), clusterRegistryConfigArgs, isHCP, cluster)
if err != nil {
    r.Reporter.Errorf("%s", err)
    os.Exit(1)
}

if clusterRegistryConfigArgs != nil {
    allowedRegistries, blockedRegistries, insecureRegistries,
        additionalTrustedCa, allowedRegistriesForImport,
        platformAllowlist := clusterregistryconfig.GetClusterRegistryConfigArgs(
        clusterRegistryConfigArgs)
        
    // NEW: Process IDMS/ITMS for edit
    imageDigestMirrorSets, imageTagMirrorSets, err := 
        clusterregistryconfig.GetImageMirrorSetArgs(clusterRegistryConfigArgs)
    if err != nil {
        r.Reporter.Errorf("Failed to parse image mirror sets: %s", err)
        os.Exit(1)
    }

    // Check if any registry config field is set (including IDMS/ITMS)
    if allowedRegistries != nil || blockedRegistries != nil || insecureRegistries != nil ||
        additionalTrustedCa != "" || allowedRegistriesForImport != "" || platformAllowlist != "" ||
        len(imageDigestMirrorSets) > 0 || len(imageTagMirrorSets) > 0 {
        
        if PromptUserToAcceptRegistryChange(r) {
            clusterConfig, err = BuildClusterConfigWithRegistry(clusterConfig, 
                allowedRegistries, blockedRegistries, insecureRegistries,
                additionalTrustedCa, allowedRegistriesForImport, platformAllowlist,
                imageDigestMirrorSets, imageTagMirrorSets) // NEW parameters
        }
        if err != nil {
            r.Reporter.Errorf("%s", err)
            os.Exit(1)
        }
    }
}
```

### 3.3 Update Command Builder
**File**: `cmd/create/cluster/cmd.go` - `buildCommand()` function around line 4180:

Add IDMS/ITMS flags to the command reconstruction:

```go
func buildCommand(spec ocm.Spec, ...) string {
    // Existing command building logic...
    
    command += clusterautoscaler.BuildAutoscalerOptions(spec.AutoscalerConfig, clusterAutoscalerFlagsPrefix)
    command += clusterregistryconfig.BuildRegistryConfigOptions(spec)
    
    // NEW: Add IDMS/ITMS options
    command += clusterregistryconfig.BuildImageMirrorSetOptions(spec)
    
    // Remaining command building...
    return command
}
```

Update `BuildRegistryConfigOptions()` in `pkg/clusterregistryconfig/flags.go`:

```go
func BuildImageMirrorSetOptions(spec ocm.Spec) string {
    command := ""

    // Build IDMS options
    if len(spec.ImageDigestMirrorSets) > 0 {
        var idmsStrings []string
        for _, idms := range spec.ImageDigestMirrorSets {
            for _, mirror := range idms.Mirrors {
                idmsString := fmt.Sprintf("%s:%s=%s", 
                    idms.Name, 
                    mirror.Source, 
                    strings.Join(mirror.MirrorsByDigest, ","))
                idmsStrings = append(idmsStrings, idmsString)
            }
        }
        if len(idmsStrings) > 0 {
            command += fmt.Sprintf(" --%s %s",
                imageDigestMirrorSetsFlag,
                shellescape.Quote(strings.Join(idmsStrings, " ")))
        }
    }

    // Build ITMS options
    if len(spec.ImageTagMirrorSets) > 0 {
        var itmsStrings []string
        for _, itms := range spec.ImageTagMirrorSets {
            for _, mirror := range itms.Mirrors {
                itmsString := fmt.Sprintf("%s:%s=%s", 
                    itms.Name, 
                    mirror.Source, 
                    strings.Join(mirror.MirrorsByTag, ","))
                itmsStrings = append(itmsStrings, itmsString)
            }
        }
        if len(itmsStrings) > 0 {
            command += fmt.Sprintf(" --%s %s",
                imageTagMirrorSetsFlag,
                shellescape.Quote(strings.Join(itmsStrings, " ")))
        }
    }

    return command
}
```

## Phase 4: Validation and Safety

### 4.1 Add Validation Functions
**File**: `pkg/clusterregistryconfig/validation.go` (new file)

```go
package clusterregistryconfig

import (
    "fmt"
    "net/url"
    "regexp"
    "strings"
    
    "github.com/openshift/rosa/pkg/ocm"
)

const (
    maxMirrorSets     = 100
    maxMirrorsPerSet  = 50
    minOpenShiftVersion = "4.13"
)

var (
    // Kubernetes naming convention validation
    kubernetesNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?$`)
    
    // Platform registries that should be protected
    platformRegistries = []string{
        "quay.io",
        "registry.redhat.io", 
        "registry.access.redhat.com",
        "registry.connect.redhat.com",
        "cloud.redhat.com",
        "console.redhat.com",
    }
)

func validateImageMirrorSetFormat(input interface{}) error {
    switch v := input.(type) {
    case string:
        return validateSingleMirrorSetFormat(v)
    case []string:
        for _, entry := range v {
            if err := validateSingleMirrorSetFormat(entry); err != nil {
                return err
            }
        }
    default:
        return fmt.Errorf("unsupported input type for mirror set validation")
    }
    return nil
}

func validateSingleMirrorSetFormat(entry string) error {
    if entry == "" {
        return nil // Empty entries are allowed and will be filtered out
    }
    
    // Parse format: "name:source=mirror1,mirror2"
    parts := strings.SplitN(entry, ":", 2)
    if len(parts) != 2 {
        return fmt.Errorf("invalid format: %s (expected 'name:source=mirror1,mirror2')", entry)
    }
    
    name := parts[0]
    if !kubernetesNameRegex.MatchString(name) {
        return fmt.Errorf("invalid name '%s': must follow Kubernetes naming conventions", name)
    }
    
    sourceMirrors := strings.SplitN(parts[1], "=", 2)
    if len(sourceMirrors) != 2 {
        return fmt.Errorf("invalid source=mirrors format: %s", parts[1])
    }
    
    source := sourceMirrors[0]
    if err := validateRegistryFormat(source); err != nil {
        return fmt.Errorf("invalid source registry '%s': %v", source, err)
    }
    
    mirrors := strings.Split(sourceMirrors[1], ",")
    if len(mirrors) > maxMirrorsPerSet {
        return fmt.Errorf("too many mirrors (%d) for mirror set '%s', maximum is %d", 
            len(mirrors), name, maxMirrorsPerSet)
    }
    
    for _, mirror := range mirrors {
        mirror = strings.TrimSpace(mirror)
        if mirror == "" {
            return fmt.Errorf("empty mirror specified for source %s", source)
        }
        if err := validateRegistryFormat(mirror); err != nil {
            return fmt.Errorf("invalid mirror registry '%s': %v", mirror, err)
        }
    }
    
    return nil
}

func validateRegistryFormat(registry string) error {
    if registry == "" {
        return fmt.Errorf("registry cannot be empty")
    }
    
    // Basic URL validation (allow registry.com and registry.com/path formats)
    if strings.Contains(registry, "://") {
        return fmt.Errorf("registry should not include protocol (http:// or https://)")
    }
    
    // Try to parse as URL to validate format
    if _, err := url.Parse("https://" + registry); err != nil {
        return fmt.Errorf("invalid registry format: %v", err)
    }
    
    return nil
}

func validateImageMirrorSetLimits(idmsList []ocm.ImageDigestMirrorSet, itmsList []ocm.ImageTagMirrorSet) error {
    totalSets := len(idmsList) + len(itmsList)
    if totalSets > maxMirrorSets {
        return fmt.Errorf("too many mirror sets (%d), maximum allowed is %d", totalSets, maxMirrorSets)
    }
    
    // Check for duplicate names
    nameMap := make(map[string]bool)
    
    for _, idms := range idmsList {
        if nameMap[idms.Name] {
            return fmt.Errorf("duplicate mirror set name: %s", idms.Name)
        }
        nameMap[idms.Name] = true
        
        for _, mirror := range idms.Mirrors {
            if len(mirror.MirrorsByDigest) > maxMirrorsPerSet {
                return fmt.Errorf("too many mirrors (%d) for IDMS '%s', maximum is %d", 
                    len(mirror.MirrorsByDigest), idms.Name, maxMirrorsPerSet)
            }
        }
    }
    
    for _, itms := range itmsList {
        if nameMap[itms.Name] {
            return fmt.Errorf("duplicate mirror set name: %s", itms.Name)
        }
        nameMap[itms.Name] = true
        
        for _, mirror := range itms.Mirrors {
            if len(mirror.MirrorsByTag) > maxMirrorsPerSet {
                return fmt.Errorf("too many mirrors (%d) for ITMS '%s', maximum is %d", 
                    len(mirror.MirrorsByTag), itms.Name, maxMirrorsPerSet)
            }
        }
    }
    
    return nil
}

func validateMirrorSetsCompatibility(idmsList []ocm.ImageDigestMirrorSet, itmsList []ocm.ImageTagMirrorSet) error {
    // Check for conflicts between IDMS and ITMS source registries
    idmsSources := make(map[string]string)
    itmsSources := make(map[string]string)
    
    for _, idms := range idmsList {
        for _, mirror := range idms.Mirrors {
            idmsSources[mirror.Source] = idms.Name
        }
    }
    
    for _, itms := range itmsList {
        for _, mirror := range itms.Mirrors {
            if conflictingIDMS, exists := idmsSources[mirror.Source]; exists {
                return fmt.Errorf("source registry '%s' cannot be used in both IDMS '%s' and ITMS '%s'", 
                    mirror.Source, conflictingIDMS, itms.Name)
            }
            itmsSources[mirror.Source] = itms.Name
        }
    }
    
    return nil
}

func validateMirrorSetsAgainstPlatformRequirements(idmsList []ocm.ImageDigestMirrorSet, itmsList []ocm.ImageTagMirrorSet) error {
    // Ensure platform registries are not blocked or misconfigured
    for _, idms := range idmsList {
        for _, mirror := range idms.Mirrors {
            for _, platformReg := range platformRegistries {
                if strings.HasPrefix(mirror.Source, platformReg) {
                    return fmt.Errorf("platform registry '%s' cannot be used as source in IDMS '%s'", 
                        mirror.Source, idms.Name)
                }
                for _, mirrorReg := range mirror.MirrorsByDigest {
                    if strings.HasPrefix(mirrorReg, platformReg) {
                        return fmt.Errorf("platform registry '%s' cannot be used as mirror in IDMS '%s'", 
                            mirrorReg, idms.Name)
                    }
                }
            }
        }
    }
    
    for _, itms := range itmsList {
        for _, mirror := range itms.Mirrors {
            for _, platformReg := range platformRegistries {
                if strings.HasPrefix(mirror.Source, platformReg) {
                    return fmt.Errorf("platform registry '%s' cannot be used as source in ITMS '%s'", 
                        mirror.Source, itms.Name)
                }
                for _, mirrorReg := range mirror.MirrorsByTag {
                    if strings.HasPrefix(mirrorReg, platformReg) {
                        return fmt.Errorf("platform registry '%s' cannot be used as mirror in ITMS '%s'", 
                            mirrorReg, itms.Name)
                    }
                }
            }
        }
    }
    
    return nil
}

func validateOpenShiftVersionSupport(version string, hasIDMS bool, hasITMS bool) error {
    if !hasIDMS && !hasITMS {
        return nil // No validation needed if not using IDMS/ITMS
    }
    
    // Extract version number (handle formats like "4.13.1", "openshift-v4.13.1", etc.)
    versionNum := strings.TrimPrefix(version, "openshift-v")
    versionNum = strings.TrimPrefix(versionNum, "v")
    
    majorMinor := strings.SplitN(versionNum, ".", 3)
    if len(majorMinor) < 2 {
        return fmt.Errorf("invalid version format: %s", version)
    }
    
    if majorMinor[0] != "4" {
        return fmt.Errorf("IDMS/ITMS requires OpenShift 4.x, got version: %s", version)
    }
    
    // Simple string comparison for minor version (13, 14, 15, etc.)
    minorVersion := majorMinor[1]
    if len(minorVersion) == 1 && minorVersion[0] < '13'[0] {
        return fmt.Errorf("IDMS/ITMS requires OpenShift 4.13+, got version: %s", version)
    } else if len(minorVersion) == 2 && minorVersion < "13" {
        return fmt.Errorf("IDMS/ITMS requires OpenShift 4.13+, got version: %s", version)
    }
    
    return nil
}

// ValidateCompleteImageMirrorSetConfiguration performs comprehensive validation
func ValidateCompleteImageMirrorSetConfiguration(idmsList []ocm.ImageDigestMirrorSet, itmsList []ocm.ImageTagMirrorSet, version string) error {
    if err := validateImageMirrorSetLimits(idmsList, itmsList); err != nil {
        return err
    }
    
    if err := validateMirrorSetsCompatibility(idmsList, itmsList); err != nil {
        return err
    }
    
    if err := validateMirrorSetsAgainstPlatformRequirements(idmsList, itmsList); err != nil {
        return err
    }
    
    if err := validateOpenShiftVersionSupport(version, len(idmsList) > 0, len(itmsList) > 0); err != nil {
        return err
    }
    
    return nil
}
```

### 4.2 Integrate Validation
Update the creation and edit flows to use validation:

In `cmd/create/cluster/cmd.go`:
```go
// After parsing IDMS/ITMS, add validation
if len(imageDigestMirrorSets) > 0 || len(imageTagMirrorSets) > 0 {
    if err := clusterregistryconfig.ValidateCompleteImageMirrorSetConfiguration(
        imageDigestMirrorSets, imageTagMirrorSets, version); err != nil {
        r.Reporter.Errorf("Image mirror set validation failed: %s", err)
        os.Exit(1)
    }
}
```

## Phase 5: Display and Output Support

### 5.1 Update Describe Command
**File**: `cmd/describe/cluster/cmd.go`

Update `getClusterRegistryConfig()` around line 946 to display IDMS/ITMS information:

```go
func getClusterRegistryConfig(cluster *cmv1.Cluster, allowlist *cmv1.RegistryAllowlist) string {
    var output string
    
    // Existing registry sources logic...
    if cluster.RegistryConfig().RegistrySources() != nil {
        registryResources := cluster.RegistryConfig().RegistrySources()
        // ... existing allowed/blocked/insecure registry display ...
    }
    
    // NEW: Display IDMS
    if cluster.RegistryConfig().ImageDigestMirrorSets() != nil && len(cluster.RegistryConfig().ImageDigestMirrorSets()) > 0 {
        output = fmt.Sprintf("%s - Image Digest Mirror Sets:\n", output)
        for _, idms := range cluster.RegistryConfig().ImageDigestMirrorSets() {
            output = fmt.Sprintf("%s    - Name:                 %s\n", output, idms.Name())
            for _, mirror := range idms.Mirrors() {
                output = fmt.Sprintf("%s      Source:               %s\n", output, mirror.Source())
                output = fmt.Sprintf("%s      Mirrors (by digest):  %s\n", output, 
                    strings.Join(mirror.MirrorsByDigest(), ", "))
            }
        }
    }
    
    // NEW: Display ITMS
    if cluster.RegistryConfig().ImageTagMirrorSets() != nil && len(cluster.RegistryConfig().ImageTagMirrorSets()) > 0 {
        output = fmt.Sprintf("%s - Image Tag Mirror Sets:\n", output)
        for _, itms := range cluster.RegistryConfig().ImageTagMirrorSets() {
            output = fmt.Sprintf("%s    - Name:                 %s\n", output, itms.Name())
            for _, mirror := range itms.Mirrors() {
                output = fmt.Sprintf("%s      Source:               %s\n", output, mirror.Source())
                output = fmt.Sprintf("%s      Mirrors (by tag):     %s\n", output, 
                    strings.Join(mirror.MirrorsByTag(), ", "))
            }
        }
    }
    
    // Existing logic for allowed registries for import and platform allowlist...
    if cluster.RegistryConfig().AllowedRegistriesForImport() != nil {
        // ... existing display logic ...
    }
    
    if cluster.RegistryConfig().PlatformAllowlist().ID() != "" && allowlist != nil {
        // ... existing display logic ...
    }
    
    return output
}
```

### 5.2 Update JSON Output
Ensure IDMS/ITMS information is included in JSON output. The existing JSON marshaling should automatically include the new fields from the OCM SDK.

## Phase 6: Documentation and Examples

### 6.1 Add Help Documentation
Update command help text and examples in relevant files:

**Examples to add to cluster creation help in `cmd/create/cluster/cmd.go`:**

```go
Example: `  # Create a cluster named "mycluster"
  rosa create cluster --cluster-name=mycluster

  # Create a cluster in the us-east-2 region
  rosa create cluster --cluster-name=mycluster --region=us-east-2
  
  # Create a cluster with Image Digest Mirror Sets
  rosa create cluster --cluster-name=mycluster \
    --registry-config-image-digest-mirror-sets="company-mirrors:registry.company.com/openshift=mirror1.company.com/openshift,mirror2.company.com/openshift"
  
  # Create a cluster with Image Tag Mirror Sets
  rosa create cluster --cluster-name=mycluster \
    --registry-config-image-tag-mirror-sets="tag-mirrors:docker.io/library=mirror.company.com/docker-library"
  
  # Create a cluster with both IDMS and ITMS
  rosa create cluster --cluster-name=mycluster \
    --registry-config-image-digest-mirror-sets="digest-mirrors:registry.redhat.io/openshift4=internal.company.com/openshift4" \
    --registry-config-image-tag-mirror-sets="tag-mirrors:docker.io=internal.company.com/docker-proxy"`,
```

### 6.2 Add CLI Structure Tests
**File**: `cmd/rosa/structure_test/command_args/rosa/create/cluster/command_args.yml`

Add new flags to the structure tests around line 100:
```yaml
- name: registry-config-allowed-registries
- name: registry-config-insecure-registries  
- name: registry-config-blocked-registries
- name: registry-config-allowed-registries-for-import
- name: registry-config-platform-allowlist
- name: registry-config-additional-trusted-ca
- name: registry-config-image-digest-mirror-sets  # NEW
- name: registry-config-image-tag-mirror-sets     # NEW
```

## Phase 7: Testing Integration

### 7.1 Unit Tests
Create comprehensive unit tests:

**File**: `pkg/clusterregistryconfig/validation_test.go`
```go
package clusterregistryconfig_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    
    "github.com/openshift/rosa/pkg/clusterregistryconfig"
    "github.com/openshift/rosa/pkg/ocm"
)

var _ = Describe("IDMS/ITMS Validation", func() {
    Context("validateImageMirrorSetFormat", func() {
        It("should accept valid IDMS format", func() {
            input := "company-mirrors:registry.company.com/openshift=mirror1.company.com/openshift,mirror2.company.com/openshift"
            err := clusterregistryconfig.validateImageMirrorSetFormat(input)
            Expect(err).ToNot(HaveOccurred())
        })
        
        It("should reject invalid format", func() {
            input := "invalid-format-missing-colon"
            err := clusterregistryconfig.validateImageMirrorSetFormat(input)
            Expect(err).To(HaveOccurred())
            Expect(err.Error()).To(ContainSubstring("invalid format"))
        })
        
        It("should reject invalid Kubernetes names", func() {
            input := "Invalid_Name:registry.com=mirror.com"
            err := clusterregistryconfig.validateImageMirrorSetFormat(input)
            Expect(err).To(HaveOccurred())
            Expect(err.Error()).To(ContainSubstring("Kubernetes naming conventions"))
        })
    })
    
    Context("validateImageMirrorSetLimits", func() {
        It("should accept valid limits", func() {
            idmsList := []ocm.ImageDigestMirrorSet{
                {
                    Name: "test-idms",
                    Mirrors: []ocm.ImageMirror{
                        {
                            Source: "registry.com",
                            MirrorsByDigest: []string{"mirror1.com", "mirror2.com"},
                        },
                    },
                },
            }
            itmsList := []ocm.ImageTagMirrorSet{}
            
            err := clusterregistryconfig.validateImageMirrorSetLimits(idmsList, itmsList)
            Expect(err).ToNot(HaveOccurred())
        })
        
        It("should reject duplicate names", func() {
            idmsList := []ocm.ImageDigestMirrorSet{
                {Name: "duplicate-name"},
            }
            itmsList := []ocm.ImageTagMirrorSet{
                {Name: "duplicate-name"},
            }
            
            err := clusterregistryconfig.validateImageMirrorSetLimits(idmsList, itmsList)
            Expect(err).To(HaveOccurred())
            Expect(err.Error()).To(ContainSubstring("duplicate mirror set name"))
        })
    })
    
    Context("validateOpenShiftVersionSupport", func() {
        It("should accept OpenShift 4.13+", func() {
            err := clusterregistryconfig.validateOpenShiftVersionSupport("4.13.1", true, false)
            Expect(err).ToNot(HaveOccurred())
        })
        
        It("should reject OpenShift 4.12", func() {
            err := clusterregistryconfig.validateOpenShiftVersionSupport("4.12.1", true, false)
            Expect(err).To(HaveOccurred())
            Expect(err.Error()).To(ContainSubstring("requires OpenShift 4.13+"))
        })
    })
})
```

**File**: `pkg/clusterregistryconfig/flags_test.go`
```go
var _ = Describe("IDMS/ITMS Flag Processing", func() {
    Context("parseImageDigestMirrorSets", func() {
        It("should parse valid IDMS configuration", func() {
            input := []string{
                "company-mirrors:registry.company.com/openshift=mirror1.company.com/openshift,mirror2.company.com/openshift",
            }
            
            result, err := clusterregistryconfig.parseImageDigestMirrorSets(input)
            Expect(err).ToNot(HaveOccurred())
            Expect(result).To(HaveLen(1))
            Expect(result[0].Name).To(Equal("company-mirrors"))
            Expect(result[0].Mirrors).To(HaveLen(1))
            Expect(result[0].Mirrors[0].Source).To(Equal("registry.company.com/openshift"))
            Expect(result[0].Mirrors[0].MirrorsByDigest).To(Equal([]string{
                "mirror1.company.com/openshift",
                "mirror2.company.com/openshift",
            }))
        })
    })
})
```

### 7.2 E2E Tests
**File**: `tests/e2e/hcp_cluster_test.go`

Extend existing registry configuration tests around line 400 to include IDMS/ITMS scenarios:

```go
Context("Registry configuration with IDMS/ITMS", func() {
    It("should create and configure cluster with IDMS", func() {
        if !clusterService.IsHostedCPCluster() {
            Skip("IDMS/ITMS is only supported for hosted clusters")
        }
        
        By("Create cluster with IDMS configuration")
        clusterID := generateClusterID()
        out, err := clusterService.Create(
            clusterID, "--hosted-cp",
            "--registry-config-image-digest-mirror-sets", 
            "test-idms:registry.example.com/test=mirror.company.com/test",
            "--billing-account", config.BillingAccount,
        )
        Expect(err).ToNot(HaveOccurred())
        textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()
        Expect(textData).To(ContainSubstring("Creating cluster '%s'", clusterID))
        
        By("Wait for cluster to be ready")
        err = clusterService.WaitForClusterToBeReady(clusterID, 120)
        Expect(err).ToNot(HaveOccurred())
        
        By("Describe cluster to verify IDMS configuration")
        output, err := clusterService.DescribeCluster(clusterID)
        Expect(err).ToNot(HaveOccurred())
        Expect(output).To(ContainSubstring("Image Digest Mirror Sets"))
        Expect(output).To(ContainSubstring("test-idms"))
        Expect(output).To(ContainSubstring("registry.example.com/test"))
        Expect(output).To(ContainSubstring("mirror.company.com/test"))
        
        defer clusterService.CleanResources(clusterID)
    })
    
    It("should edit cluster IDMS/ITMS configuration", func() {
        if !clusterService.IsHostedCPCluster() {
            Skip("IDMS/ITMS is only supported for hosted clusters")
        }
        
        By("Edit cluster to add ITMS configuration")
        out, err := clusterService.EditCluster(clusterID,
            "--registry-config-image-tag-mirror-sets", 
            "test-itms:docker.io/library=mirror.company.com/docker",
            "-y",
        )
        Expect(err).ToNot(HaveOccurred())
        textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()
        Expect(textData).To(ContainSubstring("Updated cluster '%s'", clusterID))
        
        By("Verify ITMS configuration was added")
        output, err := clusterService.DescribeCluster(clusterID)
        Expect(err).ToNot(HaveOccurred())
        Expect(output).To(ContainSubstring("Image Tag Mirror Sets"))
        Expect(output).To(ContainSubstring("test-itms"))
    })
})
```

### 7.3 Test Profile Updates
**File**: `tests/utils/handler/cluster_handler.go`

Add IDMS/ITMS support to test cluster configurations around line 880:

```go
if ch.profile.ClusterConfig.RegistriesConfig && ch.profile.ClusterConfig.HCP {
    // Existing registry config logic...
    
    // NEW: Add IDMS/ITMS test configurations
    if ch.profile.ClusterConfig.ImageDigestMirrorSets {
        flags = append(flags,
            "--registry-config-image-digest-mirror-sets", 
            "test-idms:registry.example.com/test=mirror.company.com/test,mirror2.company.com/test",
        )
    }
    
    if ch.profile.ClusterConfig.ImageTagMirrorSets {
        flags = append(flags,
            "--registry-config-image-tag-mirror-sets", 
            "test-itms:docker.io/library=mirror.company.com/docker",
        )
    }
}
```

Update test profile configuration structure to include IDMS/ITMS flags.

## Implementation Timeline

### Week 1: Core Foundation
- **Days 1-2**: Phase 1 - Core data model extensions in `pkg/ocm/`
- **Days 3-5**: Phase 2 - CLI flag extensions and parsing in `pkg/clusterregistryconfig/`

### Week 2: Command Integration
- **Days 1-3**: Phase 3 - Update create/edit cluster commands
- **Days 4-5**: Phase 4 - Validation and safety checks

### Week 3: Display and Documentation
- **Days 1-2**: Phase 5 - Update describe command and output formatting
- **Days 3-5**: Phase 6 - Documentation, help text, and examples

### Week 4: Testing and Polish
- **Days 1-3**: Phase 7 - Comprehensive testing (unit and E2E)
- **Days 4-5**: Bug fixes, refinement, and final testing

## Key Considerations

### 1. Backward Compatibility
- All existing registry configuration functionality must continue to work
- No breaking changes to existing CLI flags or behavior
- IDMS/ITMS should work alongside legacy `RegistrySources`

### 2. Version Compatibility  
- Add OpenShift version checks (4.13+ requirement for IDMS/ITMS)
- Clear error messages for unsupported versions
- Graceful handling of version detection failures

### 3. Error Handling
- Comprehensive validation with clear error messages
- Prevent platform registry misconfiguration
- Helpful suggestions for fixing configuration issues

### 4. Feature Toggle Support
- Consider adding feature flag support for gradual rollout
- Allow disabling IDMS/ITMS in certain environments if needed
- Environment-based configuration options

### 5. Platform Safety
- Maintain protection for platform registries (quay.io, registry.redhat.io, etc.)
- Validate against accidental platform registry blocking
- Comprehensive safety checks for enterprise environments

### 6. Performance Considerations
- Efficient parsing of complex mirror set configurations
- Minimal impact on cluster creation/edit performance
- Appropriate caching of validation results

### 7. User Experience
- Intuitive CLI flag naming and structure
- Clear help documentation with practical examples
- Interactive mode support for guided configuration

## Success Criteria

1. **Feature Completeness**: Full IDMS/ITMS support matching backend capabilities
2. **Safety**: Comprehensive validation preventing misconfigurations
3. **Usability**: Intuitive CLI interface with good documentation
4. **Testing**: >90% test coverage for new functionality
5. **Performance**: No significant impact on cluster operations
6. **Compatibility**: Seamless integration with existing registry features

## Risk Mitigation

1. **Configuration Complexity**: Provide clear examples and validation
2. **Platform Impact**: Extensive testing in safe environments first
3. **User Adoption**: Comprehensive documentation and migration guides
4. **Backend Changes**: Close coordination with backend API team
5. **Regression Risk**: Thorough testing of existing functionality

This plan provides a complete roadmap for implementing IDMS/ITMS support in the ROSA CLI while maintaining the high standards of safety, usability, and reliability expected from the platform. 