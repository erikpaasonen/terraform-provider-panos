// Package eth is the client.Network.EthernetInterface namespace.
//
// Normalized object:  Entry
package eth

import (
    "fmt"
    "encoding/xml"

    "github.com/PaloAltoNetworks/pango/util"
    "github.com/PaloAltoNetworks/pango/version"
)


// Entry is a normalized, version independent representation of an ethernet
// interface.
type Entry struct {
    Name string
    Mode string
    StaticIps []string
    EnableDhcp bool
    CreateDhcpDefaultRoute bool
    DhcpDefaultRouteMetric int
    Ipv6Enabled bool
    ManagementProfile string
    Mtu int
    AdjustTcpMss bool
    NetflowProfile string
    LldpEnabled bool
    LldpProfile string
    LinkSpeed string
    LinkDuplex string
    LinkState string
    AggregateGroup string
    Comment string
    Ipv4MssAdjust int
    Ipv6MssAdjust int

    raw map[string] string
}

// Copy copies the information from source Entry `s` to this object.  As the
// Name field relates to the XPATH of this object, this field is not copied.
func (o *Entry) Copy(s Entry) {
    o.Mode = s.Mode
    o.StaticIps = s.StaticIps
    o.EnableDhcp = s.EnableDhcp
    o.CreateDhcpDefaultRoute = s.CreateDhcpDefaultRoute
    o.DhcpDefaultRouteMetric = s.DhcpDefaultRouteMetric
    o.Ipv6Enabled = s.Ipv6Enabled
    o.ManagementProfile = s.ManagementProfile
    o.Mtu = s.Mtu
    o.AdjustTcpMss = s.AdjustTcpMss
    o.NetflowProfile = s.NetflowProfile
    o.LldpEnabled = s.LldpEnabled
    o.LldpProfile = s.LldpProfile
    o.LinkSpeed = s.LinkSpeed
    o.LinkDuplex = s.LinkDuplex
    o.LinkState = s.LinkState
    o.AggregateGroup = s.AggregateGroup
    o.Comment = s.Comment
    o.Ipv4MssAdjust = s.Ipv4MssAdjust
    o.Ipv6MssAdjust = s.Ipv6MssAdjust
}

// Eth is the client.Network.EthernetInterface namespace.
type Eth struct {
    con util.XapiClient
}

// Initialize is invoked by client.Initialize().
func (c *Eth) Initialize(con util.XapiClient) {
    c.con = con
}

// ShowList performs SHOW to retrieve a list of ethernet interfaces.
func (c *Eth) ShowList() ([]string, error) {
    c.con.LogQuery("(show) list of ethernet interfaces")
    path := c.xpath(nil)
    return c.con.EntryListUsing(c.con.Show, path[:len(path) - 1])
}

// GetList performs GET to retrieve a list of ethernet interfaces.
func (c *Eth) GetList() ([]string, error) {
    c.con.LogQuery("(get) list of ethernet interfaces")
    path := c.xpath(nil)
    return c.con.EntryListUsing(c.con.Get, path[:len(path) - 1])
}

// Get performs GET to retrieve information for the given ethernet interface.
func (c *Eth) Get(name string) (Entry, error) {
    c.con.LogQuery("(get) ethernet interface %q", name)
    return c.details(c.con.Get, name)
}

// Show performs SHOW to retrieve information for the given ethernet interface.
func (c *Eth) Show(name string) (Entry, error) {
    c.con.LogQuery("(show) ethernet interface %q", name)
    return c.details(c.con.Show, name)
}

// Set performs SET to create / update one or more ethernet interfaces.
//
// Specifying a non-empty vsys will import the interfaces into that vsys,
// allowing the vsys to use them, as long as the interface does not have a
// mode of "ha" or "aggregate-group".  Interfaces that have either of those
// modes are omitted from this function's followup vsys import.
func (c *Eth) Set(vsys string, e ...Entry) error {
    var err error

    if len(e) == 0 {
        return nil
    }

    _, fn := c.versioning()
    n1 := make([]string, len(e))
    n2 := make([]string, 0, len(e))

    // Build up the struct with the given interface configs.
    d := util.BulkElement{XMLName: xml.Name{Local: "ethernet"}}
    for i := range e {
        d.Data = append(d.Data, fn(e[i]))
        n1[i] = e[i].Name
        if e[i].Mode != "ha" && e[i].Mode != "aggregate-group" {
            n2 = append(n2, e[i].Name)
        }
    }
    c.con.LogAction("(set) ethernet interfaces: %v", n1)

    // Set xpath.
    path := c.xpath(n1)
    if len(e) == 1 {
        path = path[:len(path) - 1]
    } else {
        path = path[:len(path) - 2]
    }

    // Create the interfaces.
    _, err = c.con.Set(path, d.Config(), nil, nil)
    if err != nil {
        return err
    }

    // Perform vsys import next.
    if vsys == "" || len(n2) == 0 {
        return nil
    }
    return c.con.ImportInterfaces(vsys, n2)
}

// Edit performs EDIT to create / update the specified ethernet interface.
//
// Specifying a non-empty vsys will import the interface into that vsys,
// allowing the vsys to use it, as long as the interface does not have a
// mode of "ha" or "aggregate-group".  Interfaces that have either of those
// modes are omitted from this function's followup vsys import.
func (c *Eth) Edit(vsys string, e Entry) error {
    var err error

    _, fn := c.versioning()

    c.con.LogAction("(edit) ethernet interface %q", e.Name)

    // Set xpath.
    path := c.xpath([]string{e.Name})

    // Edit the interface.
    _, err = c.con.Edit(path, fn(e), nil, nil)
    if err != nil {
        return err
    }

    // Check if we should skip the import step.
    if vsys == "" || e.Mode == "ha" || e.Mode == "aggregate-group" {
        return nil
    }

    // Perform vsys import.
    return c.con.ImportInterfaces(vsys, []string{e.Name})
}

// Delete removes the given interface(s) from the firewall.
//
// Specify a non-empty vsys to have this function remove the interface(s) from
// the vsys prior to deleting them.
//
// Interfaces can be a string or an Entry object.
func (c *Eth) Delete(vsys string, e ...interface{}) error {
    var err error

    if len(e) == 0 {
        return nil
    }

    names := make([]string, len(e))
    for i := range e {
        switch v := e[i].(type) {
        case string:
            names[i] = v
        case Entry:
            names[i] = v.Name
        default:
            return fmt.Errorf("Unknown type sent to delete: %s", v)
        }
    }
    c.con.LogAction("(delete) ethernet interface(s): %v", names)

    // Unimport interfaces from the given vsys.
    err = c.con.UnimportInterfaces(vsys, names)
    if err != nil {
        return err
    }

    // Remove interfaces next.
    path := c.xpath(names)
    _, err = c.con.Delete(path, nil, nil)
    return err
}

/** Internal functions for the Eth struct **/

func (c *Eth) versioning() (normalizer, func(Entry) (interface{})) {
    v := c.con.Versioning()

    if v.Gte(version.Number{7, 1, 0, ""}) {
        return &container_v2{}, specify_v2
    } else {
        return &container_v1{}, specify_v1
    }
}

func (c *Eth) details(fn util.Retriever, name string) (Entry, error) {
    path := c.xpath([]string{name})
    obj, _ := c.versioning()
    if _, err := fn(path, nil, obj); err != nil {
        return Entry{}, err
    }
    ans := obj.Normalize()

    return ans, nil
}

func (c *Eth) xpath(vals []string) []string {
    return []string{
        "config",
        "devices",
        util.AsEntryXpath([]string{"localhost.localdomain"}),
        "network",
        "interface",
        "ethernet",
        util.AsEntryXpath(vals),
    }
}

/** Structs / functions for this namespace. **/

type normalizer interface {
    Normalize() Entry
}

type container_v1 struct {
    Answer entry_v1 `xml:"result>entry"`
}

func (o *container_v1) Normalize() Entry {
    ans := Entry{
        Name: o.Answer.Name,
        LinkSpeed: o.Answer.LinkSpeed,
        LinkDuplex: o.Answer.LinkDuplex,
        LinkState: o.Answer.LinkState,
        Comment: o.Answer.Comment,
    }
    ans.raw = make(map[string] string)
    switch {
        case o.Answer.ModeL3 != nil:
            ans.Mode = "layer3"
            ans.Ipv6Enabled = util.AsBool(o.Answer.ModeL3.Ipv6.Enabled)
            ans.ManagementProfile = o.Answer.ModeL3.ManagementProfile
            ans.Mtu = o.Answer.ModeL3.Mtu
            ans.NetflowProfile = o.Answer.ModeL3.NetflowProfile
            ans.AdjustTcpMss = util.AsBool(o.Answer.ModeL3.AdjustTcpMss)
            ans.StaticIps = util.EntToStr(o.Answer.ModeL3.StaticIps)
            if o.Answer.ModeL3.Dhcp != nil {
                ans.EnableDhcp = util.AsBool(o.Answer.ModeL3.Dhcp.Enable)
                ans.CreateDhcpDefaultRoute = util.AsBool(o.Answer.ModeL3.Dhcp.CreateDefaultRoute)
                ans.DhcpDefaultRouteMetric = o.Answer.ModeL3.Dhcp.Metric
            }
            if o.Answer.ModeL3.Arp != nil {
                ans.raw["arp"] = util.CleanRawXml(o.Answer.ModeL3.Arp.Text)
            }
            if o.Answer.ModeL3.Subinterface != nil {
                ans.raw["l3subinterface"] = util.CleanRawXml(o.Answer.ModeL3.Subinterface.Text)
            }
            if o.Answer.ModeL3.Ipv6.Address != nil {
                ans.raw["ipv6"] = util.CleanRawXml(o.Answer.ModeL3.Ipv6.Address.Text)
            }
        case o.Answer.ModeL2 != nil:
            ans.Mode = "layer2"
            ans.LldpEnabled = util.AsBool(o.Answer.ModeL2.LldpEnabled)
            ans.LldpProfile = o.Answer.ModeL2.LldpProfile
            ans.NetflowProfile = o.Answer.ModeL2.NetflowProfile
            if o.Answer.ModeL2.Subinterface != nil {
                ans.raw["l2subinterface"] = util.CleanRawXml(o.Answer.ModeL2.Subinterface.Text)
            }
        case o.Answer.ModeVwire != nil:
            ans.Mode = "virtual-wire"
            ans.LldpEnabled = util.AsBool(o.Answer.ModeVwire.LldpEnabled)
            ans.LldpProfile = o.Answer.ModeVwire.LldpProfile
            ans.NetflowProfile = o.Answer.ModeVwire.NetflowProfile
        case o.Answer.TapMode != nil:
            ans.Mode = "tap"
        case o.Answer.HaMode != nil:
            ans.Mode = "ha"
        case o.Answer.DecryptMirrorMode != nil:
            ans.Mode = "decrypt-mirror"
        case o.Answer.AggregateGroupMode != nil:
            ans.Mode = "aggregate-group"
    }

    if len(ans.raw) == 0 {
        ans.raw = nil
    }
    return ans
}

type entry_v1 struct {
    XMLName xml.Name `xml:"entry"`
    Name string `xml:"name,attr"`
    ModeL2 *otherMode `xml:"layer2"`
    ModeL3 *l3Mode_v1 `xml:"layer3"`
    ModeVwire *otherMode `xml:"virtual-wire"`
    TapMode *emptyMode `xml:"tap"`
    HaMode *emptyMode `xml:"ha"`
    DecryptMirrorMode *emptyMode `xml:"decrypt-mirror"`
    AggregateGroupMode *emptyMode `xml:"aggregate-group"`
    LinkSpeed string `xml:"link-speed,omitempty"`
    LinkDuplex string `xml:"link-duplex,omitempty"`
    LinkState string `xml:"link-state,omitempty"`
    Comment string `xml:"comment"`
}

type emptyMode struct {}

type otherMode struct {
    LldpEnabled string `xml:"lldp>enable"`
    LldpProfile string `xml:"lldp>profile"`
    NetflowProfile string `xml:"netflow-profile,omitempty"`
    Subinterface *util.RawXml `xml:"units"`
}

type l3Mode_v1 struct {
    Ipv6 ipv6 `xml:"ipv6"`
    ManagementProfile string `xml:"interface-management-profile,omitempty"`
    Mtu int `xml:"mtu,omitempty"`
    NetflowProfile string `xml:"netflow-profile,omitempty"`
    AdjustTcpMss string `xml:"adjust-tcp-mss"`
    StaticIps *util.EntryType `xml:"ip"`
    Dhcp *dhcpSettings `xml:"dhcp-client"`
    Arp *util.RawXml `xml:"arp"`
    Subinterface *util.RawXml `xml:"units"`
}

type ipv6 struct {
    Enabled string `xml:"enabled"`
    Address *util.RawXml `xml:"address"`
}

type dhcpSettings struct {
    Enable string `xml:"enable"`
    CreateDefaultRoute string `xml:"create-default-route"`
    Metric int `xml:"default-route-metric,omitempty"`
}

type container_v2 struct {
    Answer entry_v2 `xml:"result>entry"`
}

func (o *container_v2) Normalize() Entry {
    ans := Entry{
        Name: o.Answer.Name,
        LinkSpeed: o.Answer.LinkSpeed,
        LinkDuplex: o.Answer.LinkDuplex,
        LinkState: o.Answer.LinkState,
        Comment: o.Answer.Comment,
    }
    ans.raw = make(map[string] string)
    switch {
        case o.Answer.ModeL3 != nil:
            ans.Mode = "layer3"
            ans.Ipv6Enabled = util.AsBool(o.Answer.ModeL3.Ipv6.Enabled)
            ans.ManagementProfile = o.Answer.ModeL3.ManagementProfile
            ans.Mtu = o.Answer.ModeL3.Mtu
            ans.NetflowProfile = o.Answer.ModeL3.NetflowProfile
            ans.AdjustTcpMss = util.AsBool(o.Answer.ModeL3.AdjustTcpMss)
            ans.Ipv4MssAdjust = o.Answer.ModeL3.Ipv4MssAdjust
            ans.Ipv6MssAdjust = o.Answer.ModeL3.Ipv6MssAdjust
            ans.StaticIps = util.EntToStr(o.Answer.ModeL3.StaticIps)
            if o.Answer.ModeL3.Dhcp != nil {
                ans.EnableDhcp = util.AsBool(o.Answer.ModeL3.Dhcp.Enable)
                ans.CreateDhcpDefaultRoute = util.AsBool(o.Answer.ModeL3.Dhcp.CreateDefaultRoute)
                ans.DhcpDefaultRouteMetric = o.Answer.ModeL3.Dhcp.Metric
            }
            if o.Answer.ModeL3.Arp != nil {
                ans.raw["arp"] = util.CleanRawXml(o.Answer.ModeL3.Arp.Text)
            }
            if o.Answer.ModeL3.Subinterface != nil {
                ans.raw["l3subinterface"] = util.CleanRawXml(o.Answer.ModeL3.Subinterface.Text)
            }
            if o.Answer.ModeL3.Ipv6.Address != nil {
                ans.raw["ipv6"] = util.CleanRawXml(o.Answer.ModeL3.Ipv6.Address.Text)
            }
        case o.Answer.ModeL2 != nil:
            ans.Mode = "layer2"
            ans.LldpEnabled = util.AsBool(o.Answer.ModeL2.LldpEnabled)
            ans.LldpProfile = o.Answer.ModeL2.LldpProfile
            ans.NetflowProfile = o.Answer.ModeL2.NetflowProfile
            if o.Answer.ModeL2.Subinterface != nil {
                ans.raw["l2subinterface"] = util.CleanRawXml(o.Answer.ModeL2.Subinterface.Text)
            }
        case o.Answer.ModeVwire != nil:
            ans.Mode = "virtual-wire"
            ans.LldpEnabled = util.AsBool(o.Answer.ModeVwire.LldpEnabled)
            ans.LldpProfile = o.Answer.ModeVwire.LldpProfile
            ans.NetflowProfile = o.Answer.ModeVwire.NetflowProfile
        case o.Answer.TapMode != nil:
            ans.Mode = "tap"
        case o.Answer.HaMode != nil:
            ans.Mode = "ha"
        case o.Answer.DecryptMirrorMode != nil:
            ans.Mode = "decrypt-mirror"
        case o.Answer.AggregateGroupMode != nil:
            ans.Mode = "aggregate-group"
    }

    if len(ans.raw) == 0 {
        ans.raw = nil
    }
    return ans
}

type entry_v2 struct {
    XMLName xml.Name `xml:"entry"`
    Name string `xml:"name,attr"`
    ModeL3 *l3Mode_v2 `xml:"layer3"`
    ModeL2 *otherMode `xml:"layer2"`
    ModeVwire *otherMode `xml:"virtual-wire"`
    TapMode *emptyMode `xml:"tap"`
    HaMode *emptyMode `xml:"ha"`
    DecryptMirrorMode *emptyMode `xml:"decrypt-mirror"`
    AggregateGroupMode *emptyMode `xml:"aggregate-group"`
    LinkSpeed string `xml:"link-speed,omitempty"`
    LinkDuplex string `xml:"link-duplex,omitempty"`
    LinkState string `xml:"link-state,omitempty"`
    Comment string `xml:"comment"`
}

type l3Mode_v2 struct {
    Ipv6 ipv6 `xml:"ipv6"`
    ManagementProfile string `xml:"interface-management-profile,omitempty"`
    Mtu int `xml:"mtu,omitempty"`
    NetflowProfile string `xml:"netflow-profile,omitempty"`
    AdjustTcpMss string `xml:"adjust-tcp-mss>enable"`
    Ipv4MssAdjust int `xml:"adjust-tcp-mss>ipv4-mss-adjustment,omitempty"`
    Ipv6MssAdjust int `xml:"adjust-tcp-mss>ipv6-mss-adjustment,omitempty"`
    StaticIps *util.EntryType `xml:"ip"`
    Dhcp *dhcpSettings `xml:"dhcp-client"`
    Arp *util.RawXml `xml:"arp"`
    Subinterface *util.RawXml `xml:"units"`
}

func specify_v1(e Entry) interface{} {
    ans := entry_v1{
        Name: e.Name,
        LinkSpeed: e.LinkSpeed,
        LinkDuplex: e.LinkDuplex,
        LinkState: e.LinkState,
        Comment: e.Comment,
    }

    switch e.Mode {
    case "layer3":
        i := &l3Mode_v1{
            StaticIps: util.StrToEnt(e.StaticIps),
            ManagementProfile: e.ManagementProfile,
            Mtu: e.Mtu,
            NetflowProfile: e.NetflowProfile,
            AdjustTcpMss: util.YesNo(e.AdjustTcpMss),
        }
        i.Ipv6.Enabled = util.YesNo(e.Ipv6Enabled)
        if e.EnableDhcp || e.CreateDhcpDefaultRoute || e.DhcpDefaultRouteMetric != 0 {
            i.Dhcp = &dhcpSettings{
                Enable: util.YesNo(e.EnableDhcp),
                CreateDefaultRoute: util.YesNo(e.CreateDhcpDefaultRoute),
                Metric: e.DhcpDefaultRouteMetric,
            }
        }
        if text, present := e.raw["arp"]; present {
            i.Arp = &util.RawXml{text}
        }
        if text, present := e.raw["l3subinterface"]; present {
            i.Subinterface = &util.RawXml{text}
        }
        if text, present := e.raw["ipv6"]; present {
            i.Ipv6.Address = &util.RawXml{text}
        }
        ans.ModeL3 = i
    case "layer2":
        i := &otherMode{
            LldpEnabled: util.YesNo(e.LldpEnabled),
            LldpProfile: e.LldpProfile,
            NetflowProfile: e.NetflowProfile,
        }
        if text, present := e.raw["l2subinterface"]; present {
            i.Subinterface = &util.RawXml{text}
        }
        ans.ModeL2 = i
    case "virtual-wire":
        i := &otherMode{
            LldpEnabled: util.YesNo(e.LldpEnabled),
            LldpProfile: e.LldpProfile,
            NetflowProfile: e.NetflowProfile,
        }
        ans.ModeVwire = i
    case "tap":
        ans.TapMode = &emptyMode{}
    case "ha":
        ans.HaMode = &emptyMode{}
    case "decrypt-mirror":
        ans.DecryptMirrorMode = &emptyMode{}
    case "aggregate-group":
        ans.AggregateGroupMode = &emptyMode{}
    }

    return ans
}

func specify_v2(e Entry) interface{} {
    ans := entry_v2{
        Name: e.Name,
        LinkSpeed: e.LinkSpeed,
        LinkDuplex: e.LinkDuplex,
        LinkState: e.LinkState,
        Comment: e.Comment,
    }

    switch e.Mode {
    case "layer3":
        i := &l3Mode_v2{
            StaticIps: util.StrToEnt(e.StaticIps),
            ManagementProfile: e.ManagementProfile,
            Mtu: e.Mtu,
            NetflowProfile: e.NetflowProfile,
            AdjustTcpMss: util.YesNo(e.AdjustTcpMss),
            Ipv4MssAdjust: e.Ipv4MssAdjust,
            Ipv6MssAdjust: e.Ipv6MssAdjust,
        }
        i.Ipv6.Enabled = util.YesNo(e.Ipv6Enabled)
        if e.EnableDhcp || e.CreateDhcpDefaultRoute || e.DhcpDefaultRouteMetric != 0 {
            i.Dhcp = &dhcpSettings{
                Enable: util.YesNo(e.EnableDhcp),
                CreateDefaultRoute: util.YesNo(e.CreateDhcpDefaultRoute),
                Metric: e.DhcpDefaultRouteMetric,
            }
        }
        if text, present := e.raw["arp"]; present {
            i.Arp = &util.RawXml{text}
        }
        if text, present := e.raw["l3subinterface"]; present {
            i.Subinterface = &util.RawXml{text}
        }
        if text, present := e.raw["ipv6"]; present {
            i.Ipv6.Address = &util.RawXml{text}
        }
        ans.ModeL3 = i
    case "layer2":
        i := &otherMode{
            LldpEnabled: util.YesNo(e.LldpEnabled),
            LldpProfile: e.LldpProfile,
            NetflowProfile: e.NetflowProfile,
        }
        if text, present := e.raw["l2subinterface"]; present {
            i.Subinterface = &util.RawXml{text}
        }
        ans.ModeL2 = i
    case "virtual-wire":
        i := &otherMode{
            LldpEnabled: util.YesNo(e.LldpEnabled),
            LldpProfile: e.LldpProfile,
            NetflowProfile: e.NetflowProfile,
        }
        ans.ModeVwire = i
    case "tap":
        ans.TapMode = &emptyMode{}
    case "ha":
        ans.HaMode = &emptyMode{}
    case "decrypt-mirror":
        ans.DecryptMirrorMode = &emptyMode{}
    case "aggregate-group":
        ans.AggregateGroupMode = &emptyMode{}
    }

    return ans
}
