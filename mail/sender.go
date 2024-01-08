package mail

import (
	"fmt"
	"net/smtp"

	"github.com/jordan-wright/email"
)

/*
定义了GmailSender结构体：它包含发送者的姓名、邮箱地址和密码。

实现了EmailSender接口：GmailSender有一个SendEmail方法，
该方法接收邮件主题、内容、收件人列表、抄送列表、密送列表和附件列表作为参数。

构造邮件：使用github.com/jordan-wright/email包来构建邮件内容，
包括发件人、主题、正文、收件人、抄送、密送和附件。

发送邮件：使用smtp包和PlainAuth进行SMTP认证，并调用e.Send方法发送邮件。

这个GmailSender的实现允许你使用一个Gmail账户通过SMTP来发送电子邮件，
它可以用在需要邮件通知功能的Go应用程序中，如用户注册后发送验证邮件。
*/
const (
	smtpAuthAddress   = "smtp.gmail.com"
	smtpServerAddress = "smtp.gmail.com:587"
)

type EmailSender interface {
	SendEmail(
		subject string,
		content string,
		to []string,
		cc []string,
		bcc []string,
		attachFiles []string,
	) error
}

type GmailSender struct {
	name              string
	fromEmailAddress  string
	fromEmailPassword string
}

func NewGmailSender(name string, fromEmailAddress string, fromEmailPassword string) EmailSender {
	return &GmailSender{
		name:              name,
		fromEmailAddress:  fromEmailAddress,
		fromEmailPassword: fromEmailPassword,
	}
}

func (sender *GmailSender) SendEmail(
	subject string,
	content string,
	to []string,
	cc []string,
	bcc []string,
	attachFiles []string,
) error {
	e := email.NewEmail()
	e.From = fmt.Sprintf("%s <%s>", sender.name, sender.fromEmailAddress)
	e.Subject = subject
	e.HTML = []byte(content)
	e.To = to
	e.Cc = cc
	e.Bcc = bcc

	for _, f := range attachFiles {
		_, err := e.AttachFile(f)
		if err != nil {
			return fmt.Errorf("failed to attach file %s: %w", f, err)
		}
	}

	smtpAuth := smtp.PlainAuth("", sender.fromEmailAddress, sender.fromEmailPassword, smtpAuthAddress)
	return e.Send(smtpServerAddress, smtpAuth)
}

/*
这两段代码展示了如何在Go中设置和使用Gmail SMTP服务来发送电子邮件。关键步骤包括：

构建邮件发送者并配置SMTP认证。
创建电子邮件内容，包括格式化的HTML。
将电子邮件发送到指定的收件人，并支持抄送、密送和附件。
通过单元测试确保发送功能按预期工作。

知识点包括：

使用Go标准库中的net/smtp进行邮件发送。
使用第三方库github.com/jordan-wright/email简化邮件构建过程。
理解SMTP认证和连接过程。
使用测试断言来验证功能正确性。

自己新建一个gmail账号，然后在安全性下面有一个两步验证，把它搞了
搞好之后有一个叫做应用专用密码，新建一个服务名字，就能产生一个16位的随机数字了
*/
