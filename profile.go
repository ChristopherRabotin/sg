package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/jmcvetta/randutil"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Profile stores the whole test profile
type Profile struct {
	Name      string        `xml:"name,attr"`
	UID       string        `xml:"uid,attr"`
	UserAgent string        `xml:"user-agent,attr"`
	Tests     []*StressTest `xml:"test"`
}

// Validate confirms that a profile is valid and sets the parent to all children requests.
func (p *Profile) Validate() error {
	// Let's set the parent requests on all children.
	for _, test := range p.Tests {
		if test.Requests == nil || len(test.Requests) == 0 {
			return fmt.Errorf("error loading profile %s: there are no requests to send\n", profileFile)
		}

		for _, request := range test.Requests {
			request.Validate()
			if request.FwdCookies {
				log.Warning("using parent cookies in top request has no effect")
			}
			setParentRequest(request, request.Children)
		}
	}
	return nil
}

// StressTest stores the one stress test.
type StressTest struct {
	Name        string     `xml:"name,attr"`     // Name of this test.
	Description string     `xml:"description"`   // Description of this test.
	CriticalTh  Duration   `xml:"critical,attr"` // Duration above the critical level.
	WarningTh   Duration   `xml:"warning,attr"`  // Duration above the warning level.
	Requests    []*Request `xml:"request"`       // Top-level requests for this test.
	Result      []*Result  `xml:"result"`        // Test results, populated only after the tests run.
}

func (t StressTest) String() string {
	return fmt.Sprintf("%s (critical=%s, warning=%s)", t.Name, t.CriticalTh.String(), t.WarningTh.String())
}

// URL handles URL generation based on the requested pattern.
type URL struct {
	Base   string      `xml:"base,attr"`
	Tokens *[]URLToken `xml:"token"`
}

// Validate confirms the validity of a URL.
func (u *URL) Validate() {
	if u.Tokens != nil {
		for _, tok := range *u.Tokens {
			tok.Validate()
			if !strings.Contains(u.Base, tok.Token) {
				panic(fmt.Errorf("cannot find token `%s` in base %s", tok.Token, u.Base))
			}
		}
	}
}

// Generate returns a new URL based on the base and the tokens.
func (u URL) Generate() (url string) {
	url = u.Base
	if u.Tokens != nil {
		for _, tok := range *u.Tokens {
			url = strings.Replace(url, tok.Token, tok.Generate(), -1)
		}
	}
	return
}

// String implements the Stringer interface.
func (u URL) String() (url string) {
	url = u.Base
	if u.Tokens != nil {
		for _, tok := range *u.Tokens {
			url = strings.Replace(url, tok.Token, tok.String(), -1)
		}
	}
	return
}

// URLToken handles the generate of tokens for the URL.
type URLToken struct {
	Token   string `xml:"token,attr"`
	Choices string `xml:"choices,attr"`
	Pattern string `xml:"pattern,attr"`
	Min     int    `xml:"min,attr"`
	Max     int    `xml:"max,attr"`
}

// Validate checks that the definition of this token is met, and panics otherwise.
func (t URLToken) Validate() {
	if t.Token == "" {
		panic("empty token in URL definition")
	}
	if t.Choices == "" && t.Pattern == "" {
		panic("URL Token is missing both Choices and Pattern")
	}
	if t.Choices != "" {
		if !strings.Contains(t.Choices, "|") {
			panic(fmt.Errorf("choices %s does not contain any separator (|)", t.Choices))
		}
		if t.Min != 0 || t.Max != 0 {
			log.Warning("min and max definitions have no effect in URL Tokens of type Choice")
		}
	}
	if t.Pattern != "" {
		if t.Pattern != "num" && t.Pattern != "alpha" && t.Pattern != "alphanum" {
			panic(fmt.Errorf("unknown pattern %s in URL Token", t.Pattern))
		}
		if t.Min < 0 || t.Max < 0 {
			panic("min or max is negative in URL Token")
		}
		if t.Min > t.Max {
			panic("min definition is greater than max definition in URL Token")
		}
	}
}

// Generate returns a new value for a token according to the definition.
func (t URLToken) Generate() (r string) {
	if t.Choices != "" {
		r, _ = randutil.ChoiceString(strings.Split(t.Choices, "|"))
	}
	switch t.Pattern {
	case "alpha":
		r, _ = randutil.StringRange(t.Min, t.Max, randutil.Alphabet)
	case "alphanum":
		r, _ = randutil.AlphaStringRange(t.Min, t.Max)
	case "num":
		rInt, _ := randutil.IntRange(t.Min, t.Max)
		r = strconv.FormatInt(int64(rInt), 10)
	}
	return
}

func (t URLToken) String() string {
	if t.Choices != "" {
		return "(" + t.Choices + ")"
	}
	switch t.Pattern {
	case "alpha":
		return fmt.Sprintf("[A-Za-z]{%d,%d}", t.Min, t.Max)
	case "alphanum":
		return fmt.Sprintf("[A-Za-z0-9]{%d,%d}", t.Min, t.Max)
	case "num":
		return fmt.Sprintf("[0-9]{%d,%d}", len(strconv.Itoa(t.Min)), len(strconv.Itoa(t.Max)))
	}
	return "" // can't happen
}

// Tokenized stores the data handling from a given response.
type Tokenized struct {
	Response string `xml:"responseToken,attr"`
	Header   string `xml:"headerToken,attr"`
	Cookie   string `xml:"cookieToken,attr"`
	Data     string `xml:",innerxml"`
}

// IsUsed returns whether this Tokenized will be computed.
func (t Tokenized) IsUsed() bool {
	return (t.Cookie != "" || t.Header != "" || t.Response != "")
}

// Format returns the tokenized's data from a given response.
// Note: this does not use a pointer to not overwrite the initial Data.
func (t Tokenized) Format(resp *Response) (formatted string) {
	formatted = t.Data
	if !t.IsUsed() {
		return
	}
	if resp == nil {
		log.Warning("Nothing to format for %s: response is nil.", t)
		return
	}
	if t.Cookie != "" {
		// Setting the data from the cookies.
		cookies := map[string]string{}
		for _, cookie := range resp.cookies {
			cookies[cookie.Name] = cookie.Value
		}
		re := regexp.MustCompile(fmt.Sprintf("(?:%s/)([A-Za-z0-9-_]+)", t.Cookie))
		for _, match := range re.FindAllStringSubmatch(formatted, -1) {
			formatted = strings.Replace(formatted, match[0], cookies[match[1]], -1)
		}
	}
	if t.Header != "" {
		// Setting the data from the header.
		re := regexp.MustCompile(fmt.Sprintf("(?:%s/)([A-Za-z0-9-_]+)", t.Header))
		for _, match := range re.FindAllStringSubmatch(formatted, -1) {
			formatted = strings.Replace(formatted, match[0], resp.header.Get(match[1]), -1)
		}
	}
	if t.Response != "" {
		re := regexp.MustCompile(fmt.Sprintf("(?:%s/)([A-Za-z0-9-_]+)", t.Response))
		for _, match := range re.FindAllStringSubmatch(formatted, -1) {
			var value string
			err := json.Unmarshal(resp.JSON[match[1]], &value)
			if err != nil {
				log.Warning(fmt.Sprintf("could not convert response JSON field `%s` to a string: %s | json = %s", match[1], err, resp.JSON))
			}
			formatted = strings.Replace(formatted, match[0], string(value), -1)
		}
	}
	return
}

func (t Tokenized) String() string {
	s := "{Tokenized"
	if t.Cookie != "" {
		s += " with cookie"
	}
	if t.Header != "" {
		s += " with header"
	}
	if t.Data != "" {
		s += " with data"
	}
	s += "}"
	return s
}

// loadProfile loads a profile XML file.
func loadProfile(profileFile string) error {
	if profileFile == "" {
		return errors.New("profile filename is empty")
	}
	profileData, err := ioutil.ReadFile(profileFile)
	if err != nil {
		return fmt.Errorf("error loading profile %s: %s\n", profileFile, err)
	}
	p := Profile{}
	if err = xml.Unmarshal(profileData, &p); err != nil {
		return fmt.Errorf("error loading profile %s: %s\n", profileFile, err)
	}

	profile = &p
	if err = p.Validate(); err != nil {
		return err
	}

	return nil
}

// saveResult persists the results as XML.
func saveResult(profile *Profile, profileFile string) string {
	// Let's move the top result from the request to the StressTest.
	for _, test := range profile.Tests {
		test.Result = make([]*Result, len(test.Requests))
		for i, req := range test.Requests {
			req.Result.SetTimeState(test.CriticalTh.Duration, test.WarningTh.Duration)
			test.Result[i] = req.Result
		}
		test.Requests = nil
	}

	content := xmlOutputHeader() + "\n" + xmlOutputStylesheet()

	pContent, _ := xml.MarshalIndent(profile, "", "\t")
	filename := fmt.Sprintf("%s-%s.xml", strings.Replace(profileFile, ".xml", "", -1), time.Now().Format("2006-01-02_1504"))
	ioutil.WriteFile(filename, []byte(content+string(pContent)+xmlOutputFooter()), 0644)
	return filename
}

// xmlOutputHeader the header of the XML result.
func xmlOutputHeader() string {
	return `<?xml version="1.0" encoding="utf-8"?>
<?xml-stylesheet type="text/xml" href="#stylesheet"?>
<!DOCTYPE sg-result [
<!ATTLIST xsl:stylesheet
  id    ID  #REQUIRED>
]>
<sg-result>`
}

// xmlOutputFooter the footer of the XML result.
func xmlOutputFooter() string {
	return `</sg-result>`
}

// xmlOutputStylesheet returns the whole stylesheet as defined in HTMLResult.xsl.
func xmlOutputStylesheet() string {
	return `<xsl:stylesheet version="1.0" id="stylesheet"
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
	<xsl:template match="xsl:stylesheet">
		<!-- Ignores the xsl:stylesheet tags. -->
	</xsl:template>
	<xsl:template match="/sg-result/Profile">
		<html>
			<head>
				<meta http-equiv="Content-Type" content="text/html;charset=utf-8" />
				<title>
					<xsl:value-of select="concat(@name, '(UID=', @uid, ')')" />
				</title>
				<link
					href="http://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.3.5/css/bootstrap.min.css"
					rel="stylesheet" type="text/css" />
			</head>
			<body>
				<div class="container">
					<div class="row">
						<h1>Tests</h1>
						<div class="col-md-12">
							<ul>
								<xsl:apply-templates select="test" mode="toc" />
							</ul>
						</div>
					</div>
				</div>
				<xsl:apply-templates select="test" mode="detail" />
			</body>
		</html>
	</xsl:template>
	<xsl:template match="test" mode="toc">
		<li>
			<a>
				<xsl:attribute name="href">
					<xsl:value-of select="concat('#', generate-id())" />
				</xsl:attribute>
				<xsl:value-of select="@name" />
			</a>
		</li>
	</xsl:template>
	<xsl:template match="test" mode="detail">
		<div class="container">
			<h1>
				<xsl:attribute name="id">
					<xsl:value-of select="generate-id()" />
				</xsl:attribute>
				<xsl:value-of select="@name" />
			</h1>
			<div class="col-md-12">
				<p class="text-muted row">
					<xsl:value-of select="description" />
				</p>
				<p class="text-muted row">
					Critical threshold set to
					<span class="bg-danger">
						<xsl:value-of select="concat(' ', @critical, ' ')" />
					</span>
					.
					Warning threshold set to
					<span class="bg-warning">
						<xsl:value-of select="concat(' ', @warning, ' ')" />
					</span>
					.
				</p>
				<!-- Generating a table of contents. -->
				<ul>
					<xsl:apply-templates select="result" mode="toc" />
				</ul>

			</div>
			<xsl:apply-templates select="result" mode="detail" />
		</div>
	</xsl:template>
	<xsl:template match="result|spawned" mode="toc">
		<li>
			<a>
				<xsl:attribute name="href">
					<xsl:value-of select="concat('#', generate-id())" />
				</xsl:attribute>
				<xsl:value-of select="concat(@method, ' ', @url)" />
			</a>
			<ul>
				<xsl:apply-templates select="spawned" mode="toc" />
				<xsl:comment />
			</ul>
		</li>
	</xsl:template>
	<xsl:template match="result|spawned" mode="detail">
		<xsl:variable name="depth"
			select="count(ancestor::result) + count(ancestor::spawned)" />
		<!-- Let's add some padding to distinguish children requests from parents. -->
		<div>
			<xsl:attribute name="class">
		 		<xsl:value-of select="concat('col-md-', $depth)" />
			</xsl:attribute>
			<!-- Adding something to make sure this tag doesn't swallow the next div. -->
			<xsl:comment />
		</div>
		<div>
			<xsl:attribute name="class">
		 		<xsl:value-of select="concat('col-md-', 12 - $depth)" />
			</xsl:attribute>
			<xsl:element name="{concat('h', $depth + 2)}">

				<xsl:attribute name="id">
					<xsl:value-of select="generate-id()" />
				</xsl:attribute>
				<xsl:value-of select="concat(@method, ' ', @url)" />

			</xsl:element>
			<p class="text-info">
				<xsl:value-of
					select="concat(@concurrency, ' concurrent requests repeated ', @repetitions, ' time(s)')" />

				<xsl:if test="@withCookies='true'">
					with cookies
				</xsl:if>
				<xsl:if test="@withHeaders='true'">
					with custom headers
				</xsl:if>
				<xsl:if test="@withData='true'">
					with request body
				</xsl:if>
			</p>
			<h5>Status summary</h5>
			<div class="row">
				<p class="col-md-10">
					<table class="table">
						<tr>
							<th class="text-center">Status type</th>
							<td class="text-center">Errored (no response)</td>
							<td class="text-center">1xx Informational</td>
							<td class="text-center">2xx Success</td>
							<td class="text-center">3xx Redirection</td>
							<td class="text-center">4xx Client Error</td>
							<td class="text-center">5xx Server Error</td>
						</tr>
						<tr>
							<th class="text-center">Number</th>
							<td>
								<xsl:choose>
									<xsl:when test="statuses/@errored>0">
										<xsl:attribute name="class">text-danger text-center</xsl:attribute>
									</xsl:when>
									<xsl:otherwise>
										<xsl:attribute name="class">text-success text-center</xsl:attribute>
									</xsl:otherwise>
								</xsl:choose>
								<xsl:value-of select="statuses/@errored" />
							</td>
							<td class="text-info text-center">
								<xsl:value-of select="statuses/@s1xx" />
							</td>
							<td class="text-info text-center">
								<xsl:value-of select="statuses/@s2xx" />
							</td>
							<td class="text-info text-center">
								<xsl:value-of select="statuses/@s3xx" />
							</td>
							<td class="text-info text-center">
								<xsl:value-of select="statuses/@s4xx" />
							</td>
							<td class="text-info text-center">
								<xsl:value-of select="statuses/@s5xx" />
							</td>
						</tr>
					</table>
				</p>
			</div>
			<h5>Response times</h5>
			<div class="row">
				<p class="col-md-6">
					<table class="table">
						<tr>
							<xsl:for-each select="times/*[not(starts-with(local-name(), 'p'))]">
								<th class="text-center">
									<xsl:choose>
										<xsl:when test="@state='nominal'">
											<xsl:attribute name="class">bg-success text-center</xsl:attribute>
										</xsl:when>
										<xsl:when test="@state='warning'">
											<xsl:attribute name="class">bg-warning text-center</xsl:attribute>
										</xsl:when>
										<xsl:when test="@state='critical'">
											<xsl:attribute name="class">bg-danger text-center</xsl:attribute>
										</xsl:when>
									</xsl:choose>
									<xsl:value-of select="local-name()" />
								</th>
							</xsl:for-each>
						</tr>
						<tr>
							<xsl:for-each select="times/*[not(starts-with(local-name(), 'p'))]">
								<td class="text-center">
									<xsl:choose>
										<xsl:when test="@state='nominal'">
											<xsl:attribute name="class">bg-success text-center</xsl:attribute>
										</xsl:when>
										<xsl:when test="@state='warning'">
											<xsl:attribute name="class">bg-warning text-center</xsl:attribute>
										</xsl:when>
										<xsl:when test="@state='critical'">
											<xsl:attribute name="class">bg-danger text-center</xsl:attribute>
										</xsl:when>
									</xsl:choose>
									<xsl:value-of select="@duration" />
								</td>
							</xsl:for-each>
						</tr>
					</table>
				</p>
			</div>
			<div class="row">
				<p class="col-md-10">
					<table class="table">
						<tr>
							<xsl:for-each select="times/*[starts-with(local-name(), 'p')]">
								<th class="text-center">
									<xsl:choose>
										<xsl:when test="@state='nominal'">
											<xsl:attribute name="class">bg-success text-center</xsl:attribute>
										</xsl:when>
										<xsl:when test="@state='warning'">
											<xsl:attribute name="class">bg-warning text-center</xsl:attribute>
										</xsl:when>
										<xsl:when test="@state='critical'">
											<xsl:attribute name="class">bg-danger text-center</xsl:attribute>
										</xsl:when>
									</xsl:choose>
									<xsl:value-of select="local-name()" />
								</th>
							</xsl:for-each>
						</tr>
						<tr>
							<xsl:for-each select="times/*[starts-with(local-name(), 'p')]">
								<td class="text-center">
									<xsl:choose>
										<xsl:when test="@state='nominal'">
											<xsl:attribute name="class">bg-success text-center</xsl:attribute>
										</xsl:when>
										<xsl:when test="@state='warning'">
											<xsl:attribute name="class">bg-warning text-center</xsl:attribute>
										</xsl:when>
										<xsl:when test="@state='critical'">
											<xsl:attribute name="class">bg-danger text-center</xsl:attribute>
										</xsl:when>
									</xsl:choose>
									<xsl:value-of select="@duration" />
								</td>
							</xsl:for-each>
						</tr>
					</table>
				</p>
			</div>
			<h5>Status breakdown</h5>
			<div class="row">
				<p class="col-md-3">
					<table class="table table-hover">
						<thead>
							<tr>
								<th class="text-center">Status code</th>
								<th class="text-center">Number</th>
							</tr>
						</thead>
						<tbody>
							<xsl:for-each select="status">
								<tr>
									<td class="text-center">
										<xsl:value-of select="@code" />
									</td>
									<td class="text-center">
										<xsl:value-of select="@number" />
									</td>
								</tr>
							</xsl:for-each>
						</tbody>
					</table>
				</p>
			</div>
			<xsl:apply-templates select="spawned" mode="detail" />
		</div>
	</xsl:template>
</xsl:stylesheet>`
}
