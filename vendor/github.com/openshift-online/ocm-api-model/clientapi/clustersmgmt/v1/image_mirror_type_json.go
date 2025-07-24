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


package v1 // github.com/openshift-online/ocm-api-model/clientapi/clustersmgmt/v1

import (
jsoniter "github.com/json-iterator/go"
"github.com/openshift-online/ocm-api-model/clientapi/helpers"
"sort"
"fmt"
"io"
"time"
)

		// MarshalImageMirror writes a value of the 'image_mirror' type to the given writer.
		func MarshalImageMirror(object *ImageMirror, writer io.Writer) error {
			stream := helpers.NewStream(writer)
			WriteImageMirror(object, stream)
			err := stream.Flush()
			if err != nil {
				return err
			}
			return stream.Error
		}
		// WriteImageMirror writes a value of the 'image_mirror' type to the given stream.
		func WriteImageMirror(object *ImageMirror, stream *jsoniter.Stream) {
			count := 0
			stream.WriteObjectStart()
				var present_ bool
						present_ = len(object.fieldSet_) > 0 && object.fieldSet_[0] && object.mirrorsByDigest != nil
					if present_ {
						if count > 0 {
							stream.WriteMore()
						}
						stream.WriteObjectField("mirrors_by_digest")
				WriteStringList(object.mirrorsByDigest, stream)
							count++
					}
						present_ = len(object.fieldSet_) > 1 && object.fieldSet_[1] && object.mirrorsByTag != nil
					if present_ {
						if count > 0 {
							stream.WriteMore()
						}
						stream.WriteObjectField("mirrors_by_tag")
				WriteStringList(object.mirrorsByTag, stream)
							count++
					}
						present_ = len(object.fieldSet_) > 2 && object.fieldSet_[2]
					if present_ {
						if count > 0 {
							stream.WriteMore()
						}
						stream.WriteObjectField("source")
			stream.WriteString(object.source)
					}
			stream.WriteObjectEnd()
		}
		// UnmarshalImageMirror reads a value of the 'image_mirror' type from the given
		// source, which can be an slice of bytes, a string or a reader.
		func UnmarshalImageMirror(source interface{}) (object *ImageMirror, err error) {
			iterator, err := helpers.NewIterator(source)
			if err != nil {
				return
			}
			object = ReadImageMirror(iterator)
			err = iterator.Error
			return
		}
		// ReadImageMirror reads a value of the 'image_mirror' type from the given iterator.
		func ReadImageMirror(iterator *jsoniter.Iterator) *ImageMirror {
			object := &ImageMirror{
				fieldSet_: make([]bool, 3),
			}
			for {
				field := iterator.ReadObject()
				if field == "" {
					break
				}
				switch field {
					case "mirrors_by_digest":
				value := ReadStringList(iterator)
						object.mirrorsByDigest = value
						object.fieldSet_[0] = true
					case "mirrors_by_tag":
				value := ReadStringList(iterator)
						object.mirrorsByTag = value
						object.fieldSet_[1] = true
					case "source":
			value := iterator.ReadString()
						object.source = value
						object.fieldSet_[2] = true
				default:
					iterator.ReadAny()
				}
			}
			return object
		}
