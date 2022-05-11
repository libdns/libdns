package nicrudns

func (client *Client) AddMx(names []string, target string, preference string, ttl string) (*Response, error) {
	request := &Request{
		RrList: &RrList{
			Rr: []*RR{},
		},
	}
	for _, name := range names {
		request.RrList.Rr = append(request.RrList.Rr, &RR{
			Name: name,
			Type: `MX`,
			Ttl:  ttl,
			Mx: &Mx{
				Preference: preference,
				Exchange: &Exchange{
					Name: target,
				},
			},
		})
	}
	return client.Add(request)

}
