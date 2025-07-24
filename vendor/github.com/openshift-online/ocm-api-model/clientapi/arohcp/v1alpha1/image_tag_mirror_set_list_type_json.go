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


package v1alpha1 // github.com/openshift-online/ocm-api-model/clientapi/arohcp/v1alpha1

import (
v1 "github.com/openshift-online/ocm-api-model/clientapi/clustersmgmt/v1"
"fmt"
"github.com/openshift-online/ocm-api-model/clientapi/helpers"
jsoniter "github.com/json-iterator/go"
"io"
"sort"
"time"
)

		// MarshalImageTagMirrorSetList writes a list of values of the 'image_tag_mirror_set' type to
		// the given writer.
		func MarshalImageTagMirrorSetList(list []*ImageTagMirrorSet, writer io.Writer) error {
			stream := helpers.NewStream(writer)
			WriteImageTagMirrorSetList(list, stream)
			err := stream.Flush()
			if err != nil {
				return err
			}
			return stream.Error
		}
		// WriteImageTagMirrorSetList writes a list of value of the 'image_tag_mirror_set' type to
		// the given stream.
		func WriteImageTagMirrorSetList(list []*ImageTagMirrorSet, stream *jsoniter.Stream) {
			stream.WriteArrayStart()
			for i, value := range list {
				if i > 0 {
					stream.WriteMore()
				}
			WriteImageTagMirrorSet(value, stream)
			}
			stream.WriteArrayEnd()
		}
		// UnmarshalImageTagMirrorSetList reads a list of values of the 'image_tag_mirror_set' type
		// from the given source, which can be a slice of bytes, a string or a reader.
		func UnmarshalImageTagMirrorSetList(source interface{}) (items []*ImageTagMirrorSet, err error) {
			iterator, err := helpers.NewIterator(source)
			if err != nil {
				return
			}
			items = ReadImageTagMirrorSetList(iterator)
			err = iterator.Error
			return
		}
		// ReadImageTagMirrorSetList reads list of values of the ''image_tag_mirror_set' type from
		// the given iterator.
		func ReadImageTagMirrorSetList(iterator *jsoniter.Iterator) []*ImageTagMirrorSet {
			list := []*ImageTagMirrorSet{}
			for iterator.ReadArray() {
			item := ReadImageTagMirrorSet(iterator)
				list = append(list, item)
			}
			return list
		}
