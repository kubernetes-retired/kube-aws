package cfnstack

import (
	"fmt"
	"regexp"
)

type Assets interface {
	Merge(Assets) Assets
	AsMap() map[assetID]Asset
	FindAssetByStackAndFileName(string, string) Asset
}

type assetsImpl struct {
	underlying map[assetID]Asset
}

type assetID struct {
	StackName string
	Filename  string
}

func NewAssetID(stack string, file string) assetID {
	return assetID{
		StackName: stack,
		Filename:  file,
	}
}

func (a assetsImpl) Merge(other Assets) Assets {
	merged := map[assetID]Asset{}

	for k, v := range a.underlying {
		merged[k] = v
	}
	for k, v := range other.AsMap() {
		merged[k] = v
	}

	return assetsImpl{
		underlying: merged,
	}
}

func (a assetsImpl) AsMap() map[assetID]Asset {
	return a.underlying
}

func (a assetsImpl) findAssetByID(id assetID) Asset {
	asset, ok := a.underlying[id]
	if !ok {
		panic(fmt.Sprintf("[bug] failed to get the asset for the id \"%s\"", id))
	}
	return asset
}

func (a assetsImpl) FindAssetByStackAndFileName(stack string, file string) Asset {
	return a.findAssetByID(NewAssetID(stack, file))
}

type AssetsBuilder interface {
	Add(filename string, content string) AssetsBuilder
	Build() Assets
}

type assetsBuilderImpl struct {
	locProvider AssetLocationProvider
	assets      map[assetID]Asset
}

func (b *assetsBuilderImpl) Add(filename string, content string) AssetsBuilder {
	loc, err := b.locProvider.locationFor(filename)
	if err != nil {
		panic(err)
	}
	b.assets[loc.ID] = Asset{
		AssetLocation: *loc,
		Content:       content,
	}
	return b
}

func (b *assetsBuilderImpl) Build() Assets {
	return assetsImpl{
		underlying: b.assets,
	}
}

func NewAssetsBuilder(stackName string, s3URI string) AssetsBuilder {
	return &assetsBuilderImpl{
		locProvider: AssetLocationProvider{
			s3URI:     s3URI,
			stackName: stackName,
		},
		assets: map[assetID]Asset{},
	}
}

type Asset struct {
	AssetLocation
	Content string
}

type AssetLocationProvider struct {
	s3URI     string
	stackName string
}

type AssetLocation struct {
	ID     assetID
	Key    string
	Bucket string
	Path   string
	URL    string
}

func newAssetLocationProvider(stackName string, s3URI string) AssetLocationProvider {
	return AssetLocationProvider{
		s3URI:     s3URI,
		stackName: stackName,
	}
}

func (p AssetLocationProvider) locationFor(filename string) (*AssetLocation, error) {
	s3URI := p.s3URI

	re := regexp.MustCompile("s3://(?P<bucket>[^/]+)/(?P<directory>.+[^/])/*$")
	matches := re.FindStringSubmatch(s3URI)

	path := fmt.Sprintf("%s/%s", p.stackName, filename)

	var bucket string
	var key string
	if len(matches) == 3 {
		bucket = matches[1]
		directory := matches[2]

		key = fmt.Sprintf("%s/%s", directory, path)
	} else {
		re := regexp.MustCompile("s3://(?P<bucket>[^/]+)/*$")
		matches := re.FindStringSubmatch(s3URI)

		if len(matches) == 2 {
			bucket = matches[1]
			key = path
		} else {
			return nil, fmt.Errorf("failed to parse s3 uri(=%s): The valid uri pattern for it is s3://mybucket/mydir or s3://mybucket", s3URI)
		}
	}

	url := fmt.Sprintf("https://s3.amazonaws.com/%s/%s", bucket, key)
	id := assetID{StackName: p.stackName, Filename: filename}

	return &AssetLocation{
		ID:     id,
		Key:    key,
		Bucket: bucket,
		Path:   path,
		URL:    url,
	}, nil
}
