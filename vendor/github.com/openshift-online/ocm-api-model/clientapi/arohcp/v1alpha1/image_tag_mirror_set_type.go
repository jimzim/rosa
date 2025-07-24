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

		// ImageTagMirrorSet represents the values of the 'image_tag_mirror_set' type.
		//
		// ImageTagMirrorSet represents a set of mirrors for tag-based image mirroring.
// Each ImageTagMirrorSet contains a list of mirrors that can serve images
// based on tag matching. This is the modern replacement for ImageContentSourcePolicy (ICSP)
// starting with OpenShift 4.13+.
		type  ImageTagMirrorSet struct {
			fieldSet_ []bool
					mirrors []*ImageMirror
					name string
		}
		// Empty returns true if the object is empty, i.e. no attribute has a value.
		func (o *ImageTagMirrorSet) Empty() bool {
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
			// Mirrors returns the value of the 'mirrors' attribute, or
			// the zero value of the type if the attribute doesn't have a value.
			//
			// Mirrors contains the mapping from a source registry to one or more mirror registries
			func (o *ImageTagMirrorSet) Mirrors() []*ImageMirror {
				if o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0] {
					return o.mirrors
				}
				return nil
			}
			// GetMirrors returns the value of the 'mirrors' attribute and
			// a flag indicating if the attribute has a value.
			//
			// Mirrors contains the mapping from a source registry to one or more mirror registries
			func (o *ImageTagMirrorSet) GetMirrors() (value []*ImageMirror, ok bool) {
				ok = o != nil && len(o.fieldSet_) > 0 && o.fieldSet_[0]
				if ok {
					value = o.mirrors
				}
				return
			}
			// Name returns the value of the 'name' attribute, or
			// the zero value of the type if the attribute doesn't have a value.
			//
			// Name is a human-readable name for the mirror set
			func (o *ImageTagMirrorSet) Name() string {
				if o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1] {
					return o.name
				}
				return ""
			}
			// GetName returns the value of the 'name' attribute and
			// a flag indicating if the attribute has a value.
			//
			// Name is a human-readable name for the mirror set
			func (o *ImageTagMirrorSet) GetName() (value string, ok bool) {
				ok = o != nil && len(o.fieldSet_) > 1 && o.fieldSet_[1]
				if ok {
					value = o.name
				}
				return
			}
		// ImageTagMirrorSetListKind is the name of the type used to represent list of objects of
		// type 'image_tag_mirror_set'.
		const ImageTagMirrorSetListKind = "ImageTagMirrorSetList"
		// ImageTagMirrorSetListLinkKind is the name of the type used to represent links to list
		// of objects of type 'image_tag_mirror_set'.
		const ImageTagMirrorSetListLinkKind = "ImageTagMirrorSetListLink"
		// ImageTagMirrorSetNilKind is the name of the type used to nil lists of objects of
		// type 'image_tag_mirror_set'.
		const ImageTagMirrorSetListNilKind = "ImageTagMirrorSetListNil"
		// ImageTagMirrorSetList is a list of values of the 'image_tag_mirror_set' type.
		type ImageTagMirrorSetList struct {
			href  string
			link  bool
			items []*ImageTagMirrorSet
		}
		// Len returns the length of the list.
		func (l *ImageTagMirrorSetList) Len() int {
			if l == nil {
				return 0
			}
			return len(l.items)
		}
		// Items sets the items of the list.
		func (l *ImageTagMirrorSetList) SetLink(link bool) {
			l.link = link
		}
		// Items sets the items of the list.
		func (l *ImageTagMirrorSetList) SetHREF(href string) {
			l.href = href
		}
		// Items sets the items of the list.
		func (l *ImageTagMirrorSetList) SetItems(items []*ImageTagMirrorSet) {
			l.items = items
		}
		// Items returns the items of the list.
		func (l *ImageTagMirrorSetList) Items() []*ImageTagMirrorSet {
			if l == nil {
				return nil
			}
			return l.items
		}
		// Empty returns true if the list is empty.
		func (l *ImageTagMirrorSetList) Empty() bool {
			return l == nil || len(l.items) == 0
		}
		// Get returns the item of the list with the given index. If there is no item with
		// that index it returns nil.
		func (l *ImageTagMirrorSetList) Get(i int) *ImageTagMirrorSet {
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
		func (l *ImageTagMirrorSetList) Slice() []*ImageTagMirrorSet {
			var slice []*ImageTagMirrorSet
			if l == nil {
				slice = make([]*ImageTagMirrorSet, 0)
			} else {
				slice = make([]*ImageTagMirrorSet, len(l.items))
				copy(slice, l.items)
			}
			return slice
		}

		// Each runs the given function for each item of the list, in order. If the function
		// returns false the iteration stops, otherwise it continues till all the elements
		// of the list have been processed.
		func (l *ImageTagMirrorSetList) Each(f func(item *ImageTagMirrorSet) bool) {
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
		func (l *ImageTagMirrorSetList) Range(f func(index int, item *ImageTagMirrorSet) bool) {
			if l == nil {
				return
			}
			for index, item := range l.items {
				if !f(index, item) {
					break
				}
			}
		}
