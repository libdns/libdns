package nicrudns

func (client *Client) AddTxt(names []string, target string, ttl string) (*Response, error) {
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
			Txt: &Txt{
				String: target,
			},
		})
	}
	return client.Add(request)

}
