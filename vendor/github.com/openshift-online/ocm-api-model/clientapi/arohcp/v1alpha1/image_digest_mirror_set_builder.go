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

		//
		// ImageDigestMirrorSet represents a set of mirrors for digest-based image mirroring.
// Each ImageDigestMirrorSet contains a list of mirrors that can serve images
// based on digest matching. This is the modern replacement for ImageContentSourcePolicy (ICSP)
// starting with OpenShift 4.13+.
		type  ImageDigestMirrorSetBuilder struct {
			fieldSet_ []bool
					mirrors []*ImageMirrorBuilder
					name string
		}
		// NewImageDigestMirrorSet creates a new builder of 'image_digest_mirror_set' objects.
		func NewImageDigestMirrorSet() *ImageDigestMirrorSetBuilder {
			return &ImageDigestMirrorSetBuilder{
				fieldSet_: make([]bool, 2),
			}
		}
		// Empty returns true if the builder is empty, i.e. no attribute has a value.
		func (b *ImageDigestMirrorSetBuilder) Empty() bool {
			if b == nil || len(b.fieldSet_) == 0 {
				return true
			}
				for _, set := range b.fieldSet_ {
					if set {
						return false
					}
				}
				return true
		}
				// Mirrors sets the value of the 'mirrors' attribute to the given values.
				//
				// 
						func (b *ImageDigestMirrorSetBuilder) Mirrors(values ...*ImageMirrorBuilder) *ImageDigestMirrorSetBuilder {
							if len(b.fieldSet_) == 0 {
								b.fieldSet_ = make([]bool, 2)
							}
							b.mirrors = make([]*ImageMirrorBuilder, len(values))
							copy(b.mirrors, values)
							b.fieldSet_[0] = true
							return b
						}
				// Name sets the value of the 'name' attribute to the given value.
				//
				// 
				func (b *ImageDigestMirrorSetBuilder) Name(value string) *ImageDigestMirrorSetBuilder {
					if len(b.fieldSet_) == 0 {
						b.fieldSet_ = make([]bool, 2)
					}
					b.name = value
						b.fieldSet_[1] = true
					return b
				}
		// Copy copies the attributes of the given object into this builder, discarding any previous values.
		func (b *ImageDigestMirrorSetBuilder) Copy(object *ImageDigestMirrorSet) *ImageDigestMirrorSetBuilder {
			if object == nil {
				return b
			}
			if len(object.fieldSet_) > 0 {
				b.fieldSet_ = make([]bool, len(object.fieldSet_))
				copy(b.fieldSet_, object.fieldSet_)
			}
					if object.mirrors != nil {
								b.mirrors = make([]*ImageMirrorBuilder, len(object.mirrors))
								for i, v := range object.mirrors {
									b.mirrors[i] = NewImageMirror().Copy(v)
								}
					} else {
						b.mirrors = nil
					}
					b.name = object.name
			return b
		}
		// Build creates a 'image_digest_mirror_set' object using the configuration stored in the builder.
		func (b *ImageDigestMirrorSetBuilder) Build() (object *ImageDigestMirrorSet, err error) {
			object = new(ImageDigestMirrorSet)
			if len(b.fieldSet_) > 0 {
				object.fieldSet_ = make([]bool, len(b.fieldSet_))
				copy(object.fieldSet_, b.fieldSet_)
			}
					if b.mirrors != nil {
								object.mirrors = make([]*ImageMirror, len(b.mirrors))
								for i, v := range b.mirrors {
									object.mirrors[i], err = v.Build()
									if err != nil {
										return
									}
								}
					}
					object.name = b.name
			return
		}
