//go:build windows

package outlook

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/jinkp/outlook-go-mcp/internal/domain"
)

func mailStoreSession(executor *COMExecutor) OutlookSession {
	if executor == nil {
		return nil
	}
	return executor.session
}

func (s *outlookMailStore) SearchEmails(ctx context.Context, params SearchEmailsParams) ([]Email, error) {
	if err := validateSearchEmailsParams(params); err != nil {
		return nil, err
	}

	maxResults := normalizeMailSearchMaxResults(params.MaxResults)
	results := make([]Email, 0, maxResults)

	err := s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		folder, err := resolveMailFolder(session, params.Folder)
		if err != nil {
			return err
		}
		defer folder.Release()

		items, err := dispatchProperty(folder, "Items")
		if err != nil {
			return err
		}
		defer items.Release()

		_, _ = oleutil.CallMethod(items, "Sort", "[ReceivedTime]", true)

		filter, err := buildMailSearchFilter(params)
		if err != nil {
			return err
		}

		restricted, err := dispatchCall(items, "Restrict", filter)
		if err != nil {
			return wrapCOMError("restrict mail items", err)
		}
		defer restricted.Release()

		count, err := intProperty(restricted, "Count")
		if err != nil {
			return err
		}

		for i := 1; i <= count && len(results) < maxResults; i++ {
			item, err := dispatchIndexedProperty(restricted, "Item", i)
			if err != nil {
				continue
			}

			record, mapErr := mapMailSummary(item)
			item.Release()
			if mapErr != nil {
				continue
			}

			results = append(results, mapMailRecordToEmail(record))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (s *outlookMailStore) GetEmail(ctx context.Context, id string) (*Email, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: email id is required", ErrInvalidParams)
	}

	var email *Email
	err := s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		item, err := getMailItemByID(session, id)
		if err != nil {
			return err
		}
		defer item.Release()

		record, err := mapMailDetails(item)
		if err != nil {
			return err
		}

		mapped := mapMailRecordToEmail(record)
		email = &mapped
		return nil
	})
	if err != nil {
		return nil, err
	}

	return email, nil
}

func (s *outlookMailStore) ListAttachments(ctx context.Context, params ListAttachmentsParams) ([]Attachment, error) {
	if strings.TrimSpace(params.EmailID) == "" {
		return nil, fmt.Errorf("%w: email id is required", ErrInvalidParams)
	}

	attachments := make([]Attachment, 0)
	err := s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		item, err := getMailItemByID(session, params.EmailID)
		if err != nil {
			return err
		}
		defer item.Release()

		records, err := listAttachmentRecords(item)
		if err != nil {
			return err
		}

		attachments = make([]Attachment, 0, len(records))
		for _, record := range records {
			attachments = append(attachments, mapAttachmentRecord(record))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return attachments, nil
}

func deleteMailItem(item *ole.IDispatch) error {
	if item == nil {
		return ErrNotConnected
	}
	_, err := oleutil.CallMethod(item, "Delete")
	return err
}

func (s *outlookMailStore) CreateDraft(ctx context.Context, params CreateDraftParams) (*Email, error) {
	if err := validateCreateDraftParams(params); err != nil {
		return nil, err
	}

	var draft *Email
	err := s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		item, err := dispatchCall(session.ole, "CreateItem", olMailItem)
		if err != nil {
			return wrapCOMError("create draft item", err)
		}
		defer item.Release()

		if err := putProperty(item, "BodyFormat", olFormatPlain); err != nil {
			return err
		}
		if err := putProperty(item, "To", strings.Join(params.To, ";")); err != nil {
			return err
		}
		if err := putProperty(item, "Subject", params.Subject); err != nil {
			return err
		}
		if err := putProperty(item, "Body", params.Body); err != nil {
			return err
		}
		if _, err := oleutil.CallMethod(item, "Save"); err != nil {
			return wrapCOMError("save draft", err)
		}

		record, err := mapMailDetails(item)
		if err != nil {
			return err
		}

		mapped := mapMailRecordToEmail(record)
		draft = &mapped
		return nil
	})
	if err != nil {
		return nil, err
	}

	return draft, nil
}

func (s *outlookMailStore) ReplyDraft(ctx context.Context, params domain.ReplyDraftParams) (*Email, error) {
	if strings.TrimSpace(params.EmailID) == "" {
		return nil, fmt.Errorf("%w: email_id is required", ErrInvalidParams)
	}
	if strings.TrimSpace(params.Body) == "" {
		return nil, fmt.Errorf("%w: body is required", ErrInvalidParams)
	}

	var draft *Email
	err := s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		item, err := getMailItemByID(session, params.EmailID)
		if err != nil {
			return err
		}
		defer item.Release()

		reply, err := dispatchCall(item, "Reply")
		if err != nil {
			return wrapCOMError("create reply draft", err)
		}
		defer reply.Release()

		if err := putProperty(reply, "Body", params.Body); err != nil {
			return err
		}
		if _, err := oleutil.CallMethod(reply, "Save"); err != nil {
			return wrapCOMError("save reply draft", err)
		}

		record, err := mapMailDetails(reply)
		if err != nil {
			return err
		}
		mapped := mapMailRecordToEmail(record)
		draft = &mapped
		return nil
	})
	if err != nil {
		return nil, err
	}
	return draft, nil
}

func (s *outlookMailStore) ForwardDraft(ctx context.Context, params domain.ForwardDraftParams) (*Email, error) {
	if strings.TrimSpace(params.EmailID) == "" {
		return nil, fmt.Errorf("%w: email_id is required", ErrInvalidParams)
	}
	if len(params.To) == 0 {
		return nil, fmt.Errorf("%w: at least one recipient is required", ErrInvalidParams)
	}

	var draft *Email
	err := s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		item, err := getMailItemByID(session, params.EmailID)
		if err != nil {
			return err
		}
		defer item.Release()

		fwd, err := dispatchCall(item, "Forward")
		if err != nil {
			return wrapCOMError("create forward draft", err)
		}
		defer fwd.Release()

		if err := putProperty(fwd, "To", strings.Join(params.To, ";")); err != nil {
			return err
		}
		if params.Body != "" {
			if err := putProperty(fwd, "Body", params.Body); err != nil {
				return err
			}
		}
		if _, err := oleutil.CallMethod(fwd, "Save"); err != nil {
			return wrapCOMError("save forward draft", err)
		}

		record, err := mapMailDetails(fwd)
		if err != nil {
			return err
		}
		mapped := mapMailRecordToEmail(record)
		draft = &mapped
		return nil
	})
	if err != nil {
		return nil, err
	}
	return draft, nil
}

func (s *outlookMailStore) MarkRead(ctx context.Context, params domain.MarkReadParams) error {
	if strings.TrimSpace(params.EmailID) == "" {
		return fmt.Errorf("%w: email_id is required", ErrInvalidParams)
	}

	return s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		item, err := getMailItemByID(session, params.EmailID)
		if err != nil {
			return err
		}
		defer item.Release()

		// CRITICAL: Outlook's COM property is UnRead (inverse of read).
		// read=true → UnRead=false; read=false → UnRead=true
		if err := putProperty(item, "UnRead", !params.Read); err != nil {
			return err
		}
		if _, err := oleutil.CallMethod(item, "Save"); err != nil {
			return wrapCOMError("save mark_read", err)
		}
		return nil
	})
}

func (s *outlookMailStore) FlagEmail(ctx context.Context, params domain.FlagEmailParams) error {
	if strings.TrimSpace(params.EmailID) == "" {
		return fmt.Errorf("%w: email_id is required", ErrInvalidParams)
	}

	return s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		item, err := getMailItemByID(session, params.EmailID)
		if err != nil {
			return err
		}
		defer item.Release()

		flagStatus := olFlagNone
		if params.Flagged {
			flagStatus = olFlagMarked
		}
		if err := putProperty(item, "FlagStatus", flagStatus); err != nil {
			return err
		}
		if _, err := oleutil.CallMethod(item, "Save"); err != nil {
			return wrapCOMError("save flag_email", err)
		}
		return nil
	})
}

func (s *outlookMailStore) MoveEmail(ctx context.Context, params domain.MoveEmailParams) error {
	if strings.TrimSpace(params.EmailID) == "" {
		return fmt.Errorf("%w: email_id is required", ErrInvalidParams)
	}
	if strings.TrimSpace(params.Folder) == "" {
		return fmt.Errorf("%w: folder is required", ErrInvalidParams)
	}

	return s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		item, err := getMailItemByID(session, params.EmailID)
		if err != nil {
			return err
		}
		defer item.Release()

		destFolder, err := resolveAnyFolder(session, params.Folder)
		if err != nil {
			return err
		}
		defer destFolder.Release()

		if _, err := oleutil.CallMethod(item, "Move", destFolder); err != nil {
			return wrapCOMError("move email", err)
		}
		return nil
	})
}

func (s *outlookMailStore) ListFolders(ctx context.Context) ([]domain.MailFolder, error) {
	var folders []domain.MailFolder

	err := s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		rootFolders, err := dispatchProperty(session.mapi, "Folders")
		if err != nil {
			return err
		}
		defer rootFolders.Release()

		rootCount, err := intProperty(rootFolders, "Count")
		if err != nil {
			return err
		}

		for i := 1; i <= rootCount; i++ {
			store, err := dispatchIndexedProperty(rootFolders, "Item", i)
			if err != nil {
				continue
			}

			storeName, _ := stringProperty(store, "Name")
			storeEntryID, _ := stringProperty(store, "EntryID")
			storeFolderType, _ := intProperty(store, "DefaultItemType")

			folders = append(folders, domain.MailFolder{
				Name:          storeName,
				EntryID:       storeEntryID,
				ParentEntryID: "",
				FolderType:    storeFolderType,
			})

			// Walk immediate children (depth-2)
			subFolders, subErr := dispatchProperty(store, "Folders")
			store.Release()
			if subErr != nil {
				continue
			}

			subCount, _ := intProperty(subFolders, "Count")
			for j := 1; j <= subCount; j++ {
				sub, err := dispatchIndexedProperty(subFolders, "Item", j)
				if err != nil {
					continue
				}
				subName, _ := stringProperty(sub, "Name")
				subEntryID, _ := stringProperty(sub, "EntryID")
				subFolderType, _ := intProperty(sub, "DefaultItemType")

				folders = append(folders, domain.MailFolder{
					Name:          subName,
					EntryID:       subEntryID,
					ParentEntryID: storeEntryID,
					FolderType:    subFolderType,
				})
				sub.Release()
			}
			subFolders.Release()
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return folders, nil
}

func (s *outlookMailStore) DownloadAttachment(ctx context.Context, params domain.DownloadAttachmentParams) (*domain.DownloadedAttachment, error) {
	if strings.TrimSpace(params.EmailID) == "" {
		return nil, fmt.Errorf("%w: email_id is required", ErrInvalidParams)
	}
	if strings.TrimSpace(params.AttachmentID) == "" {
		return nil, fmt.Errorf("%w: attachment_id is required", ErrInvalidParams)
	}
	if strings.TrimSpace(params.DestDir) == "" {
		return nil, fmt.Errorf("%w: dest_dir is required", ErrInvalidParams)
	}
	if !filepath.IsAbs(params.DestDir) {
		return nil, fmt.Errorf("%w: dest_dir must be an absolute path", ErrInvalidParams)
	}

	var result *domain.DownloadedAttachment
	err := s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		item, err := getMailItemByID(session, params.EmailID)
		if err != nil {
			return err
		}
		defer item.Release()

		attachments, err := dispatchProperty(item, "Attachments")
		if err != nil {
			return err
		}
		defer attachments.Release()

		// Locate attachment by ID (1-based index as string)
		idx, parseErr := strconv.Atoi(params.AttachmentID)
		if parseErr != nil || idx < 1 {
			return fmt.Errorf("%w: attachment_id must be a positive integer index", ErrInvalidParams)
		}

		attachment, err := dispatchIndexedProperty(attachments, "Item", idx)
		if err != nil {
			return fmt.Errorf("%w: attachment %q", ErrNotFound, params.AttachmentID)
		}
		defer attachment.Release()

		attachName, _ := firstNonEmptyStringProperty(attachment, "FileName", "DisplayName")
		if attachName == "" {
			attachName = fmt.Sprintf("attachment-%s", params.AttachmentID)
		}

		// Triple security check
		safeName := filepath.Base(attachName) // strip any path components

		absDestDir, err := filepath.Abs(params.DestDir)
		if err != nil {
			return fmt.Errorf("%w: cannot resolve dest_dir: %v", ErrInvalidParams, err)
		}

		absPath := filepath.Join(absDestDir, safeName)

		// Verify the resolved path stays inside destDir
		if !strings.HasPrefix(absPath, absDestDir+string(os.PathSeparator)) {
			return fmt.Errorf("%w: destination path escapes dest_dir", ErrInvalidParams)
		}

		if _, err := oleutil.CallMethod(attachment, "SaveAsFile", absPath); err != nil {
			return wrapCOMError("save attachment", err)
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return fmt.Errorf("%w: stat saved file: %v", ErrCOMFailure, err)
		}

		result = &domain.DownloadedAttachment{
			Name: safeName,
			Path: absPath,
			Size: info.Size(),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *outlookMailStore) DeleteEmail(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: email id is required", ErrInvalidParams)
	}

	return s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		item, err := getMailItemByID(session, id)
		if err != nil {
			return err
		}
		defer item.Release()

		return deleteMailItem(item)
	})
}

func (s *outlookMailStore) ListEmailsInRange(ctx context.Context, params domain.ListEmailsInRangeParams) ([]domain.Email, error) {
	if params.Since.IsZero() {
		return nil, fmt.Errorf("%w: since is required", ErrInvalidParams)
	}
	if !params.Until.IsZero() && params.Since.After(params.Until) {
		return nil, fmt.Errorf("%w: since must be before until", ErrInvalidParams)
	}
	if params.MaxResults <= 0 {
		params.MaxResults = defaultMailSearchMaxResults
	}

	var emails []domain.Email
	err := s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		// Use the inbox folder as the search base for all mail
		inbox, err := dispatchCall(session.mapi, "GetDefaultFolder", olFolderInbox)
		if err != nil {
			return wrapCOMError("get inbox folder", err)
		}
		defer inbox.Release()

		items, err := dispatchProperty(inbox, "Items")
		if err != nil {
			return err
		}
		defer items.Release()

		_, _ = oleutil.CallMethod(items, "Sort", "[ReceivedTime]", true)

		filter := fmt.Sprintf("[ReceivedTime] >= '%s'", formatOutlookTime(params.Since))
		if !params.Until.IsZero() {
			filter += fmt.Sprintf(" AND [ReceivedTime] <= '%s'", formatOutlookTime(params.Until))
		}

		restricted, err := dispatchCall(items, "Restrict", filter)
		if err != nil {
			return wrapCOMError("restrict mail items by time range", err)
		}
		defer restricted.Release()

		count, err := intProperty(restricted, "Count")
		if err != nil {
			return err
		}

		for i := 1; i <= count && len(emails) < params.MaxResults; i++ {
			item, err := dispatchIndexedProperty(restricted, "Item", i)
			if err != nil {
				continue
			}
			record, mapErr := mapMailSummary(item)
			item.Release()
			if mapErr != nil {
				continue
			}
			emails = append(emails, mapMailRecordToEmail(record))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return emails, nil
}

// resolveAnyFolder walks all top-level mail stores and their immediate children (depth-2)
// to find a folder matching name (case-insensitive). Returns ErrNotFound if no match.
func resolveAnyFolder(session *outlookSession, name string) (*ole.IDispatch, error) {
	name = strings.TrimSpace(name)

	rootFolders, err := dispatchProperty(session.mapi, "Folders")
	if err != nil {
		return nil, err
	}
	defer rootFolders.Release()

	rootCount, err := intProperty(rootFolders, "Count")
	if err != nil {
		return nil, err
	}

	for i := 1; i <= rootCount; i++ {
		store, err := dispatchIndexedProperty(rootFolders, "Item", i)
		if err != nil {
			continue
		}

		// Check the top-level store itself
		storeName, nameErr := stringProperty(store, "Name")
		if nameErr == nil && strings.EqualFold(storeName, name) {
			return store, nil
		}

		// Walk immediate children
		subFolders, subErr := dispatchProperty(store, "Folders")
		store.Release()
		if subErr != nil {
			continue
		}

		subCount, _ := intProperty(subFolders, "Count")
		for j := 1; j <= subCount; j++ {
			sub, err := dispatchIndexedProperty(subFolders, "Item", j)
			if err != nil {
				continue
			}
			subName, nameErr := stringProperty(sub, "Name")
			if nameErr == nil && strings.EqualFold(subName, name) {
				subFolders.Release()
				return sub, nil
			}
			sub.Release()
		}
		subFolders.Release()
	}

	return nil, fmt.Errorf("%w: folder %q not found", ErrNotFound, name)
}

func (s *outlookMailStore) connectedSession() (*outlookSession, error) {
	session, ok := s.session.(*outlookSession)
	if !ok || session == nil || !session.IsConnected() || session.mapi == nil || session.ole == nil {
		return nil, ErrNotConnected
	}
	return session, nil
}

func resolveMailFolder(session *outlookSession, folderName string) (*ole.IDispatch, error) {
	inbox, err := dispatchCall(session.mapi, "GetDefaultFolder", olFolderInbox)
	if err != nil {
		return nil, wrapCOMError("get inbox folder", err)
	}

	folderName = strings.TrimSpace(folderName)
	if folderName == "" || strings.EqualFold(folderName, "Inbox") {
		return inbox, nil
	}

	folders, err := dispatchProperty(inbox, "Folders")
	if err != nil {
		inbox.Release()
		return nil, err
	}
	defer folders.Release()

	count, err := intProperty(folders, "Count")
	if err != nil {
		inbox.Release()
		return nil, err
	}

	for i := 1; i <= count; i++ {
		folder, err := dispatchIndexedProperty(folders, "Item", i)
		if err != nil {
			continue
		}

		name, nameErr := stringProperty(folder, "Name")
		if nameErr == nil && strings.EqualFold(name, folderName) {
			inbox.Release()
			return folder, nil
		}

		folder.Release()
	}

	slog.Default().Warn("requested Outlook folder not found; falling back to Inbox", slog.String("folder", folderName))
	return inbox, nil
}

func buildMailSearchFilter(params SearchEmailsParams) (string, error) {
	if err := validateSearchEmailsParams(params); err != nil {
		return "", err
	}

	clauses := []string{
		fmt.Sprintf("(\"urn:schemas:httpmail:subject\" LIKE '%%%s%%' OR \"urn:schemas:httpmail:textdescription\" LIKE '%%%s%%' OR \"urn:schemas:httpmail:fromemail\" LIKE '%%%s%%')", escapeDASLValue(params.Query), escapeDASLValue(params.Query), escapeDASLValue(params.Query)),
	}

	if !params.Since.IsZero() {
		clauses = append(clauses, fmt.Sprintf("[ReceivedTime] >= '%s'", formatOutlookTime(params.Since)))
	}
	if !params.Until.IsZero() {
		clauses = append(clauses, fmt.Sprintf("[ReceivedTime] <= '%s'", formatOutlookTime(params.Until)))
	}

	if len(clauses) == 1 {
		return "@SQL=" + clauses[0], nil
	}

	return "@SQL=" + strings.Join(clauses, " AND "), nil
}

func mapMailSummary(item *ole.IDispatch) (mailRecord, error) {
	id, err := stringProperty(item, "EntryID")
	if err != nil {
		return mailRecord{}, err
	}
	subject, _ := stringProperty(item, "Subject")
	from, _ := firstNonEmptyStringProperty(item, "SenderEmailAddress", "SenderName")
	toLine, _ := stringProperty(item, "To")
	ccLine, _ := stringProperty(item, "CC")
	receivedAt, _ := timeProperty(item, "ReceivedTime")
	hasAttachments, _ := boolProperty(item, "HasAttachments")

	return mailRecord{
		ID:             id,
		Subject:        subject,
		From:           from,
		To:             splitRecipients(toLine),
		CC:             splitRecipients(ccLine),
		Date:           receivedAt,
		HasAttachments: hasAttachments,
	}, nil
}

func mapMailDetails(item *ole.IDispatch) (mailRecord, error) {
	record, err := mapMailSummary(item)
	if err != nil {
		return mailRecord{}, err
	}

	record.Body, _ = stringProperty(item, "Body")
	attachments, err := listAttachmentRecords(item)
	if err != nil {
		return mailRecord{}, err
	}
	record.Attachments = attachments
	record.HasAttachments = record.HasAttachments || len(attachments) > 0
	if record.Date.IsZero() {
		record.Date, _ = firstNonZeroTimeProperty(item, "CreationTime", "LastModificationTime", "SentOn")
	}

	return record, nil
}

func listAttachmentRecords(item *ole.IDispatch) ([]attachmentRecord, error) {
	attachments, err := dispatchProperty(item, "Attachments")
	if err != nil {
		return nil, err
	}
	defer attachments.Release()

	count, err := intProperty(attachments, "Count")
	if err != nil {
		return nil, err
	}

	records := make([]attachmentRecord, 0, count)
	for i := 1; i <= count; i++ {
		attachment, err := dispatchIndexedProperty(attachments, "Item", i)
		if err != nil {
			continue
		}

		name, _ := firstNonEmptyStringProperty(attachment, "FileName", "DisplayName")
		size, _ := int64Property(attachment, "Size")
		contentType, _ := stringProperty(attachment, "Type")

		records = append(records, attachmentRecord{
			ID:          strconv.Itoa(i),
			Name:        name,
			Size:        size,
			ContentType: contentType,
		})
		attachment.Release()
	}

	return records, nil
}

func getMailItemByID(session *outlookSession, id string) (*ole.IDispatch, error) {
	item, err := dispatchCall(session.mapi, "GetItemFromID", id)
	if err != nil {
		return nil, fmt.Errorf("%w: email %q", ErrNotFound, id)
	}
	if item == nil {
		return nil, fmt.Errorf("%w: email %q", ErrNotFound, id)
	}
	return item, nil
}

func dispatchCall(disp *ole.IDispatch, method string, params ...interface{}) (*ole.IDispatch, error) {
	if disp == nil {
		return nil, ErrNotConnected
	}

	result, err := oleutil.CallMethod(disp, method, params...)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("%w: %s returned nil result", ErrCOMFailure, method)
	}

	dispatch := result.ToIDispatch()
	if dispatch == nil {
		return nil, fmt.Errorf("%w: %s returned nil dispatch", ErrCOMFailure, method)
	}

	return dispatch, nil
}

func dispatchProperty(disp *ole.IDispatch, property string, params ...interface{}) (*ole.IDispatch, error) {
	if disp == nil {
		return nil, ErrNotConnected
	}

	result, err := oleutil.GetProperty(disp, property, params...)
	if err != nil {
		return nil, wrapCOMError("get property "+property, err)
	}
	if result == nil {
		return nil, fmt.Errorf("%w: property %s returned nil result", ErrCOMFailure, property)
	}

	dispatch := result.ToIDispatch()
	if dispatch == nil {
		return nil, fmt.Errorf("%w: property %s returned nil dispatch", ErrCOMFailure, property)
	}

	return dispatch, nil
}

func dispatchIndexedProperty(disp *ole.IDispatch, property string, index int) (*ole.IDispatch, error) {
	return dispatchProperty(disp, property, index)
}

func putProperty(disp *ole.IDispatch, property string, value interface{}) error {
	if _, err := oleutil.PutProperty(disp, property, value); err != nil {
		return wrapCOMError("set property "+property, err)
	}
	return nil
}

func stringProperty(disp *ole.IDispatch, property string, params ...interface{}) (string, error) {
	value, err := scalarProperty(disp, property, params...)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(value), nil
}

func firstNonEmptyStringProperty(disp *ole.IDispatch, properties ...string) (string, error) {
	for _, property := range properties {
		value, err := stringProperty(disp, property)
		if err == nil && strings.TrimSpace(value) != "" {
			return value, nil
		}
	}
	return "", fmt.Errorf("%w: no string properties available", ErrCOMFailure)
}

func intProperty(disp *ole.IDispatch, property string, params ...interface{}) (int, error) {
	value, err := scalarProperty(disp, property, params...)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case int:
		return v, nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case uint32:
		return int(v), nil
	default:
		return 0, fmt.Errorf("%w: property %s is not an integer", ErrCOMFailure, property)
	}
}

func int64Property(disp *ole.IDispatch, property string, params ...interface{}) (int64, error) {
	value, err := scalarProperty(disp, property, params...)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint32:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("%w: property %s is not an integer", ErrCOMFailure, property)
	}
}

func boolProperty(disp *ole.IDispatch, property string, params ...interface{}) (bool, error) {
	value, err := scalarProperty(disp, property, params...)
	if err != nil {
		return false, err
	}
	boolValue, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%w: property %s is not a bool", ErrCOMFailure, property)
	}
	return boolValue, nil
}

func timeProperty(disp *ole.IDispatch, property string, params ...interface{}) (time.Time, error) {
	value, err := scalarProperty(disp, property, params...)
	if err != nil {
		return time.Time{}, err
	}
	timeValue, ok := value.(time.Time)
	if !ok {
		return time.Time{}, fmt.Errorf("%w: property %s is not a time", ErrCOMFailure, property)
	}
	return timeValue, nil
}

func firstNonZeroTimeProperty(disp *ole.IDispatch, properties ...string) (time.Time, error) {
	for _, property := range properties {
		value, err := timeProperty(disp, property)
		if err == nil && !value.IsZero() {
			return value, nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: no time properties available", ErrCOMFailure)
}

func scalarProperty(disp *ole.IDispatch, property string, params ...interface{}) (interface{}, error) {
	if disp == nil {
		return nil, ErrNotConnected
	}

	result, err := oleutil.GetProperty(disp, property, params...)
	if err != nil {
		return nil, wrapCOMError("get property "+property, err)
	}
	if result == nil {
		return nil, fmt.Errorf("%w: property %s returned nil result", ErrCOMFailure, property)
	}
	defer result.Clear()

	value := result.Value()
	if value == nil {
		return nil, fmt.Errorf("%w: property %s returned nil value", ErrCOMFailure, property)
	}
	return value, nil
}

func wrapCOMError(action string, err error) error {
	return fmt.Errorf("%w: %s: %v", ErrCOMFailure, action, err)
}

func escapeDASLValue(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "'", "''")
}

func formatOutlookTime(value time.Time) string {
	return value.Local().Format("01/02/2006 03:04 PM")
}
