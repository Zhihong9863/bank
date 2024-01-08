package mail

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/techschool/bank/util"
)

/*
定义了一个测试：TestSendEmailWithGmail是一个测试函数，用来测试GmailSender发送电子邮件的功能。

配置加载：使用util.LoadConfig来加载配置，其中包含了邮件发送者的姓名、邮箱和密码。

发送测试邮件：构造了一个测试邮件，包括主题、HTML内容、收件人和附件，并使用sender.SendEmail发送。

断言：使用require.NoError来确保发送邮件的过程中没有发生错误。

这个测试用例可以在开发过程中用来验证你的邮件发送功能是否正常工作。
*/
func TestSendEmailWithGmail(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	config, err := util.LoadConfig("..")
	require.NoError(t, err)

	sender := NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)

	subject := "A test email"
	content := `
	<h1>Hello world</h1>
	<p>This is a test message from <a href="http://techschool.guru">Tech School</a></p>
	`
	to := []string{"hezhihong98@gmail.com"}
	attachFiles := []string{"../README.md"}

	err = sender.SendEmail(subject, content, to, nil, nil, attachFiles)
	require.NoError(t, err)
}
