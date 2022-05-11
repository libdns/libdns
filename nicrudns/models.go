package nicrudns

import (
	"encoding/xml"
)

type Request struct {
	XMLName xml.Name `xml:"request"`
	Text    string   `xml:",chardata"`
	RrList  *RrList  `xml:"rr-list"`
}

type RrList struct {
	Text string `xml:",chardata"`
	Rr   []*RR  `xml:"rr"`
}

type RR struct {
	Text    string `xml:",chardata"`
	ID      string `xml:"id,attr,omitempty"`
	Name    string `xml:"name"`
	IdnName string `xml:"idn-name,omitempty"`
	Ttl     string `xml:"ttl"`
	Type    string `xml:"type"`
	Soa     *Soa   `xml:"soa"`
	A       *A     `xml:"a"`
	AAAA    *AAAA  `xml:"aaaa"`
	Cname   *Cname `xml:"cname"`
	Ns      *Ns    `xml:"ns"`
	Mx      *Mx    `xml:"mx"`
	Srv     *Srv   `xml:"srv"`
	Ptr     *Ptr   `xml:"ptr"`
	Txt     *Txt   `xml:"txt"`
	Dname   *Dname `xml:"dname"`
	Hinfo   *Hinfo `xml:"hinfo"`
	Naptr   *Naptr `xml:"naptr"`
	Rp      *Rp    `xml:"rp"`
}

type A string

func (a *A) String() string {
	return string(*a)
}

type AAAA string

func (aaaa *AAAA) String() string {
	return string(*aaaa)
}

type Service struct {
	Text         string `xml:",chardata"`
	Admin        string `xml:"admin,attr"`
	DomainsLimit string `xml:"domains-limit,attr"`
	DomainsNum   string `xml:"domains-num,attr"`
	Enable       string `xml:"enable,attr"`
	HasPrimary   string `xml:"has-primary,attr"`
	Name         string `xml:"name,attr"`
	Payer        string `xml:"payer,attr"`
	Tariff       string `xml:"tariff,attr"`
	RrLimit      string `xml:"rr-limit,attr"`
	RrNum        string `xml:"rr-num,attr"`
}

type Soa struct {
	Text    string `xml:",chardata"`
	Mname   *Mname `xml:"mname"`
	Rname   *Rname `xml:"rname"`
	Serial  string `xml:"serial"`
	Refresh string `xml:"refresh"`
	Retry   string `xml:"retry"`
	Expire  string `xml:"expire"`
	Minimum string `xml:"minimum"`
}

type Mname struct {
	Text    string `xml:",chardata"`
	Name    string `xml:"name"`
	IdnName string `xml:"idn-name,omitempty"`
}

type Rname struct {
	Text    string `xml:",chardata"`
	Name    string `xml:"name"`
	IdnName string `xml:"idn-name,omitempty"`
}

type Ns struct {
	Text    string `xml:",chardata"`
	Name    string `xml:"name"`
	IdnName string `xml:"idn-name,omitempty"`
}

type Mx struct {
	Text       string    `xml:",chardata"`
	Preference string    `xml:"preference"`
	Exchange   *Exchange `xml:"exchange"`
}

type Exchange struct {
	Text string `xml:",chardata"`
	Name string `xml:"name"`
}

type Srv struct {
	Text     string `xml:",chardata"`
	Priority string `xml:"priority"`
	Weight   string `xml:"weight"`
	Port     string `xml:"port"`
	Target   struct {
		Text string `xml:",chardata"`
		Name string `xml:"name"`
	} `xml:"target"`
}

type Ptr struct {
	Text string `xml:",chardata"`
	Name string `xml:"name"`
}

type Hinfo struct {
	Text     string `xml:",chardata"`
	Hardware string `xml:"hardware"`
	Os       string `xml:"os"`
}

type Naptr struct {
	Text        string `xml:",chardata"`
	Order       string `xml:"order"`
	Preference  string `xml:"preference"`
	Flags       string `xml:"flags"`
	Service     string `xml:"service"`
	Regexp      string `xml:"regexp"`
	Replacement struct {
		Text string `xml:",chardata"`
		Name string `xml:"name"`
	} `xml:"replacement"`
}

type Rp struct {
	Text      string `xml:",chardata"`
	MboxDname struct {
		Text string `xml:",chardata"`
		Name string `xml:"name"`
	} `xml:"mbox-dname"`
	TxtDname struct {
		Text string `xml:",chardata"`
		Name string `xml:"name"`
	} `xml:"txt-dname"`
}

type Cname struct {
	Text    string `xml:",chardata"`
	Name    string `xml:"name"`
	IdnName string `xml:"idn-name,omitempty"`
}

type Dname struct {
	Text string `xml:",chardata"`
	Name string `xml:"name"`
}

type Txt struct {
	Text   string `xml:",chardata"`
	String string `xml:"string"`
}

type Zone struct {
	Text       string `xml:",chardata"`
	Admin      string `xml:"admin,attr"`
	Enable     string `xml:"enable,attr"`
	HasChanges string `xml:"has-changes,attr"`
	HasPrimary string `xml:"has-primary,attr"`
	ID         string `xml:"id,attr"`
	IdnName    string `xml:"idn-name,attr"`
	Name       string `xml:"name,attr"`
	Payer      string `xml:"payer,attr"`
	Service    string `xml:"service,attr"`
	Rr         []*RR  `xml:"rr"`
}

type Revision struct {
	Text   string `xml:",chardata"`
	Date   string `xml:"date,attr"`
	Ip     string `xml:"ip,attr"`
	Number string `xml:"number,attr"`
}

type Error struct {
	Text string `xml:",chardata"`
	Code string `xml:"code,attr"`
}

type Response struct {
	XMLName xml.Name `xml:"response"`
	Text    string   `xml:",chardata"`
	Status  string   `xml:"status"`
	Errors  struct {
		Text  string `xml:",chardata"`
		Error *error `xml:"error"`
	} `xml:"errors"`
	Data struct {
		Text     string `xml:",chardata"`
		Service  []*Service
		Zone     []*Zone     `xml:"zone"`
		Address  []*A        `xml:"address"`
		Revision []*Revision `xml:"revision"`
	} `xml:"data"`
}
