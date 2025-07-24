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

		//
		// ImageMirror represents a mapping from a source registry to one or more mirror registries.
// It supports both digest-based and tag-based mirroring depending on the context.
		type  ImageMirrorBuilder struct {
			fieldSet_ []bool
					mirrorsByDigest []string
					mirrorsByTag []string
					source string
		}
		// NewImageMirror creates a new builder of 'image_mirror' objects.
		func NewImageMirror() *ImageMirrorBuilder {
			return &ImageMirrorBuilder{
				fieldSet_: make([]bool, 3),
			}
		}
		// Empty returns true if the builder is empty, i.e. no attribute has a value.
		func (b *ImageMirrorBuilder) Empty() bool {
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
				// MirrorsByDigest sets the value of the 'mirrors_by_digest' attribute to the given values.
				//
				// 
						func (b *ImageMirrorBuilder) MirrorsByDigest(values ...string) *ImageMirrorBuilder {
							if len(b.fieldSet_) == 0 {
								b.fieldSet_ = make([]bool, 3)
							}
							b.mirrorsByDigest = make([]string, len(values))
							copy(b.mirrorsByDigest, values)
							b.fieldSet_[0] = true
							return b
						}
				// MirrorsByTag sets the value of the 'mirrors_by_tag' attribute to the given values.
				//
				// 
						func (b *ImageMirrorBuilder) MirrorsByTag(values ...string) *ImageMirrorBuilder {
							if len(b.fieldSet_) == 0 {
								b.fieldSet_ = make([]bool, 3)
							}
							b.mirrorsByTag = make([]string, len(values))
							copy(b.mirrorsByTag, values)
							b.fieldSet_[1] = true
							return b
						}
				// Source sets the value of the 'source' attribute to the given value.
				//
				// 
				func (b *ImageMirrorBuilder) Source(value string) *ImageMirrorBuilder {
					if len(b.fieldSet_) == 0 {
						b.fieldSet_ = make([]bool, 3)
					}
					b.source = value
						b.fieldSet_[2] = true
					return b
				}
		// Copy copies the attributes of the given object into this builder, discarding any previous values.
		func (b *ImageMirrorBuilder) Copy(object *ImageMirror) *ImageMirrorBuilder {
			if object == nil {
				return b
			}
			if len(object.fieldSet_) > 0 {
				b.fieldSet_ = make([]bool, len(object.fieldSet_))
				copy(b.fieldSet_, object.fieldSet_)
			}
					if object.mirrorsByDigest != nil {
								b.mirrorsByDigest = make([]string, len(object.mirrorsByDigest))
								copy(b.mirrorsByDigest, object.mirrorsByDigest)
					} else {
						b.mirrorsByDigest = nil
					}
					if object.mirrorsByTag != nil {
								b.mirrorsByTag = make([]string, len(object.mirrorsByTag))
								copy(b.mirrorsByTag, object.mirrorsByTag)
					} else {
						b.mirrorsByTag = nil
					}
					b.source = object.source
			return b
		}

		// Build creates a 'image_mirror' object using the configuration stored in the builder.
		func (b *ImageMirrorBuilder) Build() (object *ImageMirror, err error) {
			object = new(ImageMirror)
			if len(b.fieldSet_) > 0 {
				object.fieldSet_ = make([]bool, len(b.fieldSet_))
				copy(object.fieldSet_, b.fieldSet_)
			}
					if b.mirrorsByDigest != nil {
								object.mirrorsByDigest = make([]string, len(b.mirrorsByDigest))
								copy(object.mirrorsByDigest, b.mirrorsByDigest)
					}
					if b.mirrorsByTag != nil {
								object.mirrorsByTag = make([]string, len(b.mirrorsByTag))
								copy(object.mirrorsByTag, b.mirrorsByTag)
					}
					object.source = b.source
			return
		}
