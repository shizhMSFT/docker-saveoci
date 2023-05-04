package convert

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/manifest/schema2"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/shizhMSFT/docker-saveoci/internal/tarutil"
)

// DockerToOCI converts a tarball in docker save format to OCI layout format.
func DockerToOCI(r io.Reader, ws io.WriteSeeker) error {
	// convert blobs
	var manifests []manifestEntry
	descriptors := make(map[string]distribution.Descriptor) // path -> descriptor
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		switch {
		case header.Name == "manifest.json":
			if err := json.NewDecoder(tr).Decode(&manifests); err != nil {
				return err
			}
		case strings.HasSuffix(header.Name, ".json"):
			digest := digest.NewDigestFromEncoded(digest.SHA256, header.Name[:len(header.Name)-5])
			descriptors[header.Name] = distribution.Descriptor{
				MediaType: schema2.MediaTypeImageConfig,
				Digest:    digest,
				Size:      header.Size,
			}
			header := &tar.Header{
				Name: ociBlobPath(digest),
				Size: header.Size,
			}
			if err := tarutil.Copy(ws, header.Name, header.Size, tr); err != nil {
				return err
			}
		case strings.HasSuffix(header.Name, "/layer.tar"):
			digester := digest.SHA256.Digester()
			ew, err := tarutil.NewEntryWriter(ws)
			if err != nil {
				return err
			}
			gw := gzip.NewWriter(io.MultiWriter(ew, digester.Hash()))
			if _, err := io.Copy(gw, tr); err != nil {
				return err
			}
			if err := gw.Close(); err != nil {
				return err
			}
			digest := digester.Digest()
			if err := ew.Commit(ociBlobPath(digest)); err != nil {
				return err
			}
			descriptors[header.Name] = distribution.Descriptor{
				MediaType: schema2.MediaTypeLayer,
				Digest:    digest,
				Size:      ew.Size(),
			}
		}
	}

	// convert manifests
	if len(manifests) == 0 {
		return errors.New("no image found")
	}
	index := ocispec.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2, // historical value
		},
	}
	for _, entry := range manifests {
		// convert manifest
		config := descriptors[entry.Config]
		layers := make([]distribution.Descriptor, 0, len(entry.Layers))
		for _, layer := range entry.Layers {
			layers = append(layers, descriptors[layer])
		}
		manifest, err := schema2.FromStruct(schema2.Manifest{
			Versioned: schema2.SchemaVersion,
			Config:    config,
			Layers:    layers,
		})
		if err != nil {
			return err
		}
		mediaType, payload, err := manifest.Payload()
		if err != nil {
			return err
		}
		desc := ocispec.Descriptor{
			MediaType: mediaType,
			Digest:    digest.FromBytes(payload),
			Size:      int64(len(payload)),
		}

		// write manifest
		if err := tarutil.WriteFile(ws, ociBlobPath(desc.Digest), payload); err != nil {
			return err
		}

		// update index
		for _, repoTag := range entry.RepoTags {
			if !strings.Contains(repoTag, "/") {
				repoTag = "docker.io/library/" + repoTag
			}
			_, tag, ok := strings.Cut(repoTag, ":")
			if !ok {
				return errors.New("invalid repoTag: " + repoTag)
			}
			manifestDesc := desc
			manifestDesc.Annotations = map[string]string{
				"io.containerd.image.name": repoTag,
				ocispec.AnnotationRefName:  tag,
			}
			index.Manifests = append(index.Manifests, manifestDesc)
		}
	}

	// write index
	indexJSON, err := json.Marshal(index)
	if err != nil {
		return fmt.Errorf("failed to marshal index file: %w", err)
	}
	if err := tarutil.WriteFile(ws, "index.json", indexJSON); err != nil {
		return err
	}

	// write oci-layout
	layout := ocispec.ImageLayout{
		Version: ocispec.ImageLayoutVersion,
	}
	layoutJSON, err := json.Marshal(layout)
	if err != nil {
		return fmt.Errorf("failed to marshal OCI layout file: %w", err)
	}
	if err := tarutil.WriteFile(ws, ocispec.ImageLayoutFile, layoutJSON); err != nil {
		return err
	}

	return tarutil.Close(ws)
}

// ociBlobPath returns the blob path for the OCI layout.
func ociBlobPath(digest digest.Digest) string {
	return "blobs/sha256/" + digest.Encoded()
}
