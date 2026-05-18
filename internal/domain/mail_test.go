package domain

import "testing"

// Phase-1 compile gate: ensures all new DTO types exist with the correct fields.
// These tests are structural — they verify type existence and zero-value safety.

func TestReplyDraftParamsHasRequiredFields(t *testing.T) {
	p := ReplyDraftParams{EmailID: "id-1", Body: "body"}
	if p.EmailID == "" {
		t.Fatal("EmailID is required")
	}
	if p.Body == "" {
		t.Fatal("Body is required")
	}
}

func TestForwardDraftParamsHasRequiredFields(t *testing.T) {
	p := ForwardDraftParams{EmailID: "id-1", To: []string{"a@b.com"}, Body: "fwd"}
	if p.EmailID == "" {
		t.Fatal("EmailID is required")
	}
	if len(p.To) == 0 {
		t.Fatal("To must be non-empty")
	}
}

func TestMarkReadParamsCarriesReadBool(t *testing.T) {
	p := MarkReadParams{EmailID: "id-1", Read: true}
	if !p.Read {
		t.Fatal("Read must be true")
	}
	p2 := MarkReadParams{EmailID: "id-1", Read: false}
	if p2.Read {
		t.Fatal("Read must be false")
	}
}

func TestFlagEmailParamsCarriesFlaggedBool(t *testing.T) {
	p := FlagEmailParams{EmailID: "id-1", Flagged: true}
	if !p.Flagged {
		t.Fatal("Flagged must be true")
	}
}

func TestMoveEmailParamsHasEmailIDAndFolder(t *testing.T) {
	p := MoveEmailParams{EmailID: "id-1", Folder: "Archive"}
	if p.EmailID == "" || p.Folder == "" {
		t.Fatalf("MoveEmailParams = %+v, expected non-empty fields", p)
	}
}

func TestDownloadAttachmentParamsHasAllFields(t *testing.T) {
	p := DownloadAttachmentParams{EmailID: "id-1", AttachmentID: "att-1", DestDir: "/tmp"}
	if p.EmailID == "" || p.AttachmentID == "" || p.DestDir == "" {
		t.Fatalf("DownloadAttachmentParams = %+v, expected non-empty fields", p)
	}
}

func TestDownloadedAttachmentHasNamePathSize(t *testing.T) {
	d := DownloadedAttachment{Name: "file.pdf", Path: "/tmp/file.pdf", Size: 1024}
	if d.Name == "" || d.Path == "" || d.Size == 0 {
		t.Fatalf("DownloadedAttachment = %+v, expected non-zero fields", d)
	}
}

func TestMailFolderHasAllFields(t *testing.T) {
	f := MailFolder{Name: "Inbox", EntryID: "eid-1", ParentEntryID: "peid-1", FolderType: 6}
	if f.Name == "" || f.EntryID == "" {
		t.Fatalf("MailFolder = %+v, expected non-empty Name and EntryID", f)
	}
}
