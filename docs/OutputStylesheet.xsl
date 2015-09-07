<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="1.0"
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
	<xsl:output method="xml" indent="yes"
		doctype-public="-//W3C//DTD XHTML 1.0 Strict//EN" />
	<xsl:template match="/Profile">
		<html>
			<head>
				<title>
					<xsl:value-of select="@name" />
					(UID=
					<xsl:value-of select="@uid" />
					)
				</title>
				<style>
					table{
					border:1px
					}
				</style>
			</head>
			<body>
				<xsl:apply-templates select="test" />
			</body>
		</html>
	</xsl:template>
	<xsl:template match="test">
		<h1>
			<xsl:value-of select="@name" />
		</h1>
		<div>
			<p>
				<xsl:value-of select="description" />
			</p>
			<xsl:apply-templates select="request" />
		</div>
	</xsl:template>
	<xsl:template match="request">
		<p>
			This request was repeated
			<xsl:value-of select="@repeat" />
			times with a concurrency of
			<xsl:value-of select="@concurrency" />
			.
		</p>
		<xsl:apply-templates select="result" />
	</xsl:template>
	<xsl:template match="result|spawned">
		<div class="level{count(ancestor::spawned)}">
			<span class="definition">
				<xsl:value-of select="@method" />
				&#160;
				<xsl:value-of select="@url" />
				&#215;
				<xsl:value-of select="@repetitions" />
				(concurrency =
				<xsl:value-of select="@concurrency" />
				)
				<xsl:if test="@withCookies='true'">
					with cookies
				</xsl:if>
				<xsl:if test="@withHeaders='true'">
					with custom headers
				</xsl:if>
				<xsl:if test="@withData='true'">
					with request body
				</xsl:if>
			</span>
			<p>
				<span class="hdr">Status summary</span>
				<table>
					<tr>
						<th>Status type</th>
						<td>Errored (no response)</td>
						<td>1xx Informational</td>
						<td>2xx Success</td>
						<td>3xx Redirection</td>
						<td>4xx Client Error</td>
						<td>5xx Server Error</td>
					</tr>
					<tr>
						<th>Number</th>
						<td>
							<xsl:value-of select="statuses/@errored" />
						</td>
						<td>
							<xsl:value-of select="statuses/@s1xx" />
						</td>
						<td>
							<xsl:value-of select="statuses/@s2xx" />
						</td>
						<td>
							<xsl:value-of select="statuses/@s3xx" />
						</td>
						<td>
							<xsl:value-of select="statuses/@s4xx" />
						</td>
						<td>
							<xsl:value-of select="statuses/@s5xx" />
						</td>
					</tr>
				</table>
			</p>
			<p>
				<span class="hdr">Status breakdown</span>
				<table>
					<thead>
						<tr>
							<td>Status code</td>
							<td>Number</td>
						</tr>
					</thead>
					<tbody>
						<xsl:for-each select="status">
							<tr>
								<td>
									<xsl:value-of select="@code" />
								</td>
								<td>
									<xsl:value-of select="@number" />
								</td>
							</tr>
						</xsl:for-each>
					</tbody>
				</table>
			</p>
			<xsl:apply-templates select="spawned" />
		</div>
	</xsl:template>
</xsl:stylesheet>
