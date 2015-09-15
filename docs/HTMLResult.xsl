<xsl:stylesheet version="1.0" id="stylesheet"
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
</xsl:stylesheet>
