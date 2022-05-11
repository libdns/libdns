package nicrudns

func (client *Client) AddA(names []string, target string, ttl string) (*Response, error) {
	request := &Request{
		RrList: &RrList{
			Rr: []*RR{},
		},
	}
	tgt := A(target)
	for _, name := range names {
		request.RrList.Rr = append(request.RrList.Rr, &RR{
			Name: name,
			Type: `A`,
			Ttl:  ttl,
			A:    &tgt,
		})
	}
	return client.Add(request)
}
