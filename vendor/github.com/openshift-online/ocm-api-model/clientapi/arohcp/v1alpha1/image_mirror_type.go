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
"time"
)

		// ImageMirror represents the values of the 'image_mirror' type.
		//
		// ImageMirror represents a mapping from a source registry to one or more mirror registries.
// It supports both digest-based and tag-based mirroring depending on the context.
		type  ImageMirror struct {
			fieldSet_ []bool
					mirrorsByDigest []string
					mirrorsByTag []string
					source string
		}
		// Empty returns true if the object is empty, i.e. no attribute has a value.
		func (o *ImageMirror) Empty() bool {
			if o == nil || len(o.fieldSet_) == 0 {
				return true
			}
				for _, set := range o.fieldSet_ {
					if set {
						return false
					}
				}
				return true
		}
			// MirrorsByDigest returns the value of the 'mirrors_by_digest' attribute, or
			// the zero value of the type if the attribute doesn't have a value.
			//
			// MirrorsByDigest contains one or more alternative locations that can serve the same images by digest
			func (o *ImageMirror) MirrorsByDigest() []string {
				if o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0] {
					return o.mirrorsByDigest
				}
				return nil
			}
			// GetMirrorsByDigest returns the value of the 'mirrors_by_digest' attribute and
			// a flag indicating if the attribute has a value.
			//
			// MirrorsByDigest contains one or more alternative locations that can serve the same images by digest
			func (o *ImageMirror) GetMirrorsByDigest() (value []string, ok bool) {
				ok = o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0]
				if ok {
					value = o.mirrorsByDigest
				}
				return
			}
			// MirrorsByTag returns the value of the 'mirrors_by_tag' attribute, or
			// the zero value of the type if the attribute doesn't have a value.
			//
			// MirrorsByTag contains one or more alternative locations that can serve the same images by tag
			func (o *ImageMirror) MirrorsByTag() []string {
				if o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1] {
					return o.mirrorsByTag
				}
				return nil
			}
			// GetMirrorsByTag returns the value of the 'mirrors_by_tag' attribute and
			// a flag indicating if the attribute has a value.
			//
			// MirrorsByTag contains one or more alternative locations that can serve the same images by tag
			func (o *ImageMirror) GetMirrorsByTag() (value []string, ok bool) {
				ok = o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1]
				if ok {
					value = o.mirrorsByTag
				}
				return
			}
			// Source returns the value of the 'source' attribute, or
			// the zero value of the type if the attribute doesn't have a value.
			//
			// Source is the registry location that images are pulled from
			func (o *ImageMirror) Source() string {
				if o != nil && len(o.fieldSet_) > 2 && o.fieldSet_[2] {
					return o.source
				}
				return ""
			}
			// GetSource returns the value of the 'source' attribute and
			// a flag indicating if the attribute has a value.
			//
			// Source is the registry location that images are pulled from
			func (o *ImageMirror) GetSource() (value string, ok bool) {
				ok = o != nil && len(o.fieldSet_) > 2 && o.fieldSet_[2]
				if ok {
					value = o.source
				}
				return
			}
		// ImageMirrorListKind is the name of the type used to represent list of objects of
		// type 'image_mirror'.
		const ImageMirrorListKind = "ImageMirrorList"
		// ImageMirrorListLinkKind is the name of the type used to represent links to list
		// of objects of type 'image_mirror'.
		const ImageMirrorListLinkKind = "ImageMirrorListLink"
		// ImageMirrorNilKind is the name of the type used to nil lists of objects of
		// type 'image_mirror'.
		const ImageMirrorListNilKind = "ImageMirrorListNil"
		// ImageMirrorList is a list of values of the 'image_mirror' type.
		type ImageMirrorList struct {
			href  string
			link  bool
			items []*ImageMirror
		}
		// Len returns the length of the list.
		func (l *ImageMirrorList) Len() int {
			if l == nil {
				return 0
			}
			return len(l.items)
		}
		// Items sets the items of the list.
		func (l *ImageMirrorList) SetLink(link bool) {
			l.link = link
		}
		// Items sets the items of the list.
		func (l *ImageMirrorList) SetHREF(href string) {
			l.href = href
		}
		// Items sets the items of the list.
		func (l *ImageMirrorList) SetItems(items []*ImageMirror) {
			l.items = items
		}
		// Items returns the items of the list.
		func (l *ImageMirrorList) Items() []*ImageMirror {
			if l == nil {
				return nil
			}
			return l.items
		}
		// Empty returns true if the list is empty.
		func (l *ImageMirrorList) Empty() bool {
			return l == nil || len(l.items) == 0
		}
		// Get returns the item of the list with the given index. If there is no item with
		// that index it returns nil.
		func (l *ImageMirrorList) Get(i int) *ImageMirror {
			if l == nil || i < 0 || i >= len(l.items) {
				return nil
			}
			return l.items[i]
		}
		// Slice returns an slice containing the items of the list. The returned slice is a
		// copy of the one used internally, so it can be modified without affecting the
		// internal representation.
		//
		// If you don't need to modify the returned slice consider using the Each or Range
		// functions, as they don't need to allocate a new slice.
		func (l *ImageMirrorList) Slice() []*ImageMirror {
			var slice []*ImageMirror
			if l == nil {
				slice = make([]*ImageMirror, 0)
			} else {
				slice = make([]*ImageMirror, len(l.items))
				copy(slice, l.items)
			}
			return slice
		}

		// Each runs the given function for each item of the list, in order. If the function
		// returns false the iteration stops, otherwise it continues till all the elements
		// of the list have been processed.
		func (l *ImageMirrorList) Each(f func(item *ImageMirror) bool) {
			if l == nil {
				return
			}
			for _, item := range l.items {
				if !f(item) {
					break
				}
			}
		}

		// Range runs the given function for each index and item of the list, in order. If
		// the function returns false the iteration stops, otherwise it continues till all
		// the elements of the list have been processed.
		func (l *ImageMirrorList) Range(f func(index int, item *ImageMirror) bool) {
			if l == nil {
				return
			}
			for index, item := range l.items {
				if !f(index, item) {
					break
				}
			}
		}
