namespace Tdl.Plugin.Tests;

public class CaseTests
{
	[Theory]
	[InlineData("userID", new[] { "user", "ID" })]
	[InlineData("HTTPServer", new[] { "HTTP", "Server" })]
	[InlineData("last4", new[] { "last4" })]
	[InlineData("user_id", new[] { "user", "id" })]
	[InlineData("Order", new[] { "Order" })]
	[InlineData("getHTTPResponseCode", new[] { "get", "HTTP", "Response", "Code" })]
	[InlineData("v2Name", new[] { "v2", "Name" })]
	[InlineData("__", new string[0])]
	[InlineData("ÜberName", new[] { "Über", "Name" })]
	public void SplitsWords(string name, string[] words) => Assert.Equal(words, Case.Words(name));

	[Theory]
	[InlineData("user_id", "UserId", "userId", "user_id", "USER_ID")]
	[InlineData("userID", "UserID", "userID", "user_id", "USER_ID")]
	[InlineData("HTTPServer", "HTTPServer", "httpServer", "http_server", "HTTP_SERVER")]
	[InlineData("", "", "", "", "")]
	public void Styles(string name, string pascal, string camel, string snake, string screaming)
	{
		var words = Case.Words(name);

		Assert.Equal(pascal, Case.Pascal(words));
		Assert.Equal(camel, Case.Camel(words));
		Assert.Equal(snake, Case.Snake(words));
		Assert.Equal(screaming, Case.ScreamingSnake(words));
	}

	[Theory]
	[InlineData("shop.Order", "Order")]
	[InlineData("Order", "Order")]
	public void TakesTheLastSegment(string name, string last) => Assert.Equal(last, Case.LastSegment(name));

	[Theory]
	[InlineData("IN_TRANSIT", "InTransit")]
	[InlineData("ok", "Ok")]
	[InlineData("_A__B_", "AB")]
	public void ReadsScreamingSnake(string s, string pascal) => Assert.Equal(pascal, Case.FromScreaming(s));
}
