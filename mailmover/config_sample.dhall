{ IMAPHost          = "imap.example.org:993"
, SMTPHost          = "smtp.example.org:587"
, Mailbox           = "INBOX"
, TrashMailbox      = "Papierkorb"
, From              = "banana@example.org"
, FromName          = "Bana Na"
, Login             = { Username = "banana@example.org", Password = "hunter2" }
, Rules             = [ { From = ".*", To = "apple@example.org", RewriteFrom = True } ]
, IgnoreFetchErrors = False
}
