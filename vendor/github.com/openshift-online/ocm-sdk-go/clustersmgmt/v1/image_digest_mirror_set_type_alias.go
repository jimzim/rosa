/*
Copyright (c) 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// IMPORTANT: This file has been generated automatically, refrain from modifying it manually as all
// your changes will be lost when the file is generated again.

package v1 // github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1

import (
	api_v1 "github.com/openshift-online/ocm-api-model/clientapi/clustersmgmt/v1"
)

// ImageDigestMirrorSet represents the values of the 'image_digest_mirror_set' type.
//
// ImageDigestMirrorSet represents a set of mirrors for digest-based image mirroring.
// Each ImageDigestMirrorSet contains a list of mirrors that can serve images
// based on digest matching. This is the modern replacement for ImageContentSourcePolicy (ICSP)
// starting with OpenShift 4.13+.
type ImageDigestMirrorSet = api_v1.ImageDigestMirrorSet

// ImageDigestMirrorSetListKind is the name of the type used to represent list of objects of
// type 'image_digest_mirror_set'.
const ImageDigestMirrorSetListKind = api_v1.ImageDigestMirrorSetListKind

// ImageDigestMirrorSetListLinkKind is the name of the type used to represent links to list
// of objects of type 'image_digest_mirror_set'.
const ImageDigestMirrorSetListLinkKind = api_v1.ImageDigestMirrorSetListLinkKind

// ImageDigestMirrorSetNilKind is the name of the type used to nil lists of objects of
// type 'image_digest_mirror_set'.
const ImageDigestMirrorSetListNilKind = api_v1.ImageDigestMirrorSetListNilKind

type ImageDigestMirrorSetList = api_v1.ImageDigestMirrorSetList
