# Announcement Email Broadcast Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an optional email broadcast step after saving a notice or system announcement.

**Architecture:** Keep site announcement persistence unchanged, then run an optional, separate email broadcast flow. The backend exposes a root-only broadcast endpoint that queries eligible users and sends the edited email draft. The frontend adds one shared modal used by both notice and system-announcement settings.

**Tech Stack:** Go, Gin, GORM, `common.SendEmail`, React 18, Semi UI, Bun/node source tests, `marked`.

---

## File Structure

- Create `service/announcement_email_broadcast.go`: validates broadcast input, selects recipients, sends mail, returns counts.
- Create `service/announcement_email_broadcast_test.go`: tests user targeting, skipped users, failed sends, and validation.
- Create `controller/announcement_email_broadcast.go`: root-only handler for `/api/notice/email-broadcast`.
- Modify `router/api-router.go`: register the new route under `/api/notice/email-broadcast` with `middleware.RootAuth()`.
- Create `web/src/components/settings/AnnouncementEmailBroadcastModal.jsx`: shared email-draft modal.
- Create `web/src/components/settings/AnnouncementEmailBroadcastModal.source.test.js`: source test for editable title/body/target and `marked.parse`.
- Modify `web/src/components/settings/OtherSetting.jsx`: trigger optional email broadcast after `Notice` save.
- Create `web/src/components/settings/OtherSetting.emailBroadcast.source.test.js`: source test for notice save integration.
- Modify `web/src/pages/Setting/Dashboard/SettingsAnnouncements.jsx`: track latest add/edit announcement and trigger optional email broadcast after save.
- Create `web/src/pages/Setting/Dashboard/SettingsAnnouncements.emailBroadcast.source.test.js`: source test for system-announcement integration.
- Modify locale JSON files only if extraction/lint requires keys; otherwise rely on Chinese source keys already passed to `t()`.

## Task 1: Backend Broadcast Service

**Files:**
- Create: `service/announcement_email_broadcast_test.go`
- Create: `service/announcement_email_broadcast.go`

- [ ] **Step 1: Write the failing service tests**

Create `service/announcement_email_broadcast_test.go` with:

```go
package service

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func seedBroadcastUser(t *testing.T, user model.User) model.User {
	t.Helper()
	if user.Password == "" {
		user.Password = "hashed-password"
	}
	if user.Group == "" {
		user.Group = "default"
	}
	if user.AffCode == "" {
		user.AffCode = user.Username
	}
	require.NoError(t, model.DB.Create(&user).Error)
	return user
}

func TestBroadcastAnnouncementEmailTargetsAgentsOnly(t *testing.T) {
	db := setupAdminPermissionServiceTestDB(t)
	_ = db
	seedBroadcastUser(t, model.User{Username: "broadcast_agent", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeAgent, Email: "agent@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_end_user", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeEndUser, Email: "user@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_admin", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, UserType: model.UserTypeAdmin, Email: "admin@example.com"})

	originalSender := sendAnnouncementBroadcastEmail
	t.Cleanup(func() { sendAnnouncementBroadcastEmail = originalSender })
	recipients := []string{}
	sendAnnouncementBroadcastEmail = func(subject string, receiver string, content string) error {
		recipients = append(recipients, receiver)
		require.Equal(t, "Subject", subject)
		require.Equal(t, "<p>Body</p>", content)
		return nil
	}

	result, err := BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{
		Source:  AnnouncementEmailSourceNotice,
		Target:  AnnouncementEmailTargetAgent,
		Title:   "Subject",
		Content: "<p>Body</p>",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.SentCount)
	require.Equal(t, 0, result.SkippedCount)
	require.Equal(t, 0, result.FailedCount)
	require.Equal(t, []string{"agent@example.com"}, recipients)
}

func TestBroadcastAnnouncementEmailTargetsEndUsersAndLegacyBlankUserType(t *testing.T) {
	setupAdminPermissionServiceTestDB(t)
	seedBroadcastUser(t, model.User{Username: "broadcast_end_user_current", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeEndUser, Email: "current@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_end_user_legacy", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: "", Email: "legacy@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_agent_skip", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeAgent, Email: "agent-skip@example.com"})

	originalSender := sendAnnouncementBroadcastEmail
	t.Cleanup(func() { sendAnnouncementBroadcastEmail = originalSender })
	recipients := []string{}
	sendAnnouncementBroadcastEmail = func(subject string, receiver string, content string) error {
		recipients = append(recipients, receiver)
		return nil
	}

	result, err := BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{
		Source:  AnnouncementEmailSourceAnnouncement,
		Target:  AnnouncementEmailTargetEndUser,
		Title:   "Subject",
		Content: "<p>Body</p>",
	})

	require.NoError(t, err)
	require.Equal(t, 2, result.SentCount)
	require.ElementsMatch(t, []string{"current@example.com", "legacy@example.com"}, recipients)
}

func TestBroadcastAnnouncementEmailTargetsAllNonAdminUsers(t *testing.T) {
	setupAdminPermissionServiceTestDB(t)
	seedBroadcastUser(t, model.User{Username: "broadcast_all_agent", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeAgent, Email: "agent@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_all_user", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeEndUser, Email: "user@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_all_admin", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, UserType: model.UserTypeAdmin, Email: "admin@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_all_root", Role: common.RoleRootUser, Status: common.UserStatusEnabled, UserType: model.UserTypeRoot, Email: "root@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_all_disabled", Role: common.RoleCommonUser, Status: common.UserStatusDisabled, UserType: model.UserTypeEndUser, Email: "disabled@example.com"})

	originalSender := sendAnnouncementBroadcastEmail
	t.Cleanup(func() { sendAnnouncementBroadcastEmail = originalSender })
	recipients := []string{}
	sendAnnouncementBroadcastEmail = func(subject string, receiver string, content string) error {
		recipients = append(recipients, receiver)
		return nil
	}

	result, err := BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{
		Source:  AnnouncementEmailSourceNotice,
		Target:  AnnouncementEmailTargetAll,
		Title:   "Subject",
		Content: "<p>Body</p>",
	})

	require.NoError(t, err)
	require.Equal(t, 2, result.SentCount)
	require.ElementsMatch(t, []string{"agent@example.com", "user@example.com"}, recipients)
}

func TestBroadcastAnnouncementEmailSkipsEmptyEmailAndContinuesAfterFailure(t *testing.T) {
	setupAdminPermissionServiceTestDB(t)
	seedBroadcastUser(t, model.User{Username: "broadcast_skip_no_email", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeEndUser})
	seedBroadcastUser(t, model.User{Username: "broadcast_fail_email", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeEndUser, Email: "fail@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_ok_email", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeEndUser, Email: "ok@example.com"})

	originalSender := sendAnnouncementBroadcastEmail
	t.Cleanup(func() { sendAnnouncementBroadcastEmail = originalSender })
	sendAnnouncementBroadcastEmail = func(subject string, receiver string, content string) error {
		if receiver == "fail@example.com" {
			return errors.New("smtp failed")
		}
		return nil
	}

	result, err := BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{
		Source:  AnnouncementEmailSourceNotice,
		Target:  AnnouncementEmailTargetEndUser,
		Title:   "Subject",
		Content: "<p>Body</p>",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.SentCount)
	require.Equal(t, 1, result.SkippedCount)
	require.Equal(t, 1, result.FailedCount)
}

func TestBroadcastAnnouncementEmailRejectsInvalidInput(t *testing.T) {
	setupAdminPermissionServiceTestDB(t)

	_, err := BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{Source: "bad", Target: AnnouncementEmailTargetAll, Title: "Subject", Content: "<p>Body</p>"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid source")

	_, err = BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{Source: AnnouncementEmailSourceNotice, Target: "bad", Title: "Subject", Content: "<p>Body</p>"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid target")

	_, err = BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{Source: AnnouncementEmailSourceNotice, Target: AnnouncementEmailTargetAll, Title: " ", Content: "<p>Body</p>"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "title is required")

	_, err = BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{Source: AnnouncementEmailSourceNotice, Target: AnnouncementEmailTargetAll, Title: "Subject", Content: " "})
	require.Error(t, err)
	require.Contains(t, err.Error(), "content is required")
}
```

- [ ] **Step 2: Run service tests to verify RED**

Run:

```powershell
go test ./service -run BroadcastAnnouncementEmail -count=1
```

Expected: FAIL because `BroadcastAnnouncementEmail`, request/result types, constants, and `sendAnnouncementBroadcastEmail` are undefined.

- [ ] **Step 3: Implement the service**

Create `service/announcement_email_broadcast.go`:

```go
package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	AnnouncementEmailSourceNotice       = "notice"
	AnnouncementEmailSourceAnnouncement = "announcement"
	AnnouncementEmailTargetAgent        = "agent"
	AnnouncementEmailTargetEndUser      = "end_user"
	AnnouncementEmailTargetAll          = "all"
)

type AnnouncementEmailBroadcastRequest struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type AnnouncementEmailBroadcastResult struct {
	SentCount    int `json:"sent_count"`
	SkippedCount int `json:"skipped_count"`
	FailedCount  int `json:"failed_count"`
}

var sendAnnouncementBroadcastEmail = common.SendEmail

func BroadcastAnnouncementEmail(req AnnouncementEmailBroadcastRequest) (AnnouncementEmailBroadcastResult, error) {
	req.Source = strings.TrimSpace(req.Source)
	req.Target = strings.TrimSpace(req.Target)
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if req.Source != AnnouncementEmailSourceNotice && req.Source != AnnouncementEmailSourceAnnouncement {
		return AnnouncementEmailBroadcastResult{}, fmt.Errorf("invalid source: %s", req.Source)
	}
	if req.Target != AnnouncementEmailTargetAgent && req.Target != AnnouncementEmailTargetEndUser && req.Target != AnnouncementEmailTargetAll {
		return AnnouncementEmailBroadcastResult{}, fmt.Errorf("invalid target: %s", req.Target)
	}
	if req.Title == "" {
		return AnnouncementEmailBroadcastResult{}, errors.New("title is required")
	}
	if req.Content == "" {
		return AnnouncementEmailBroadcastResult{}, errors.New("content is required")
	}

	users, err := listAnnouncementEmailRecipients(req.Target)
	if err != nil {
		return AnnouncementEmailBroadcastResult{}, err
	}

	result := AnnouncementEmailBroadcastResult{}
	for _, user := range users {
		email := strings.TrimSpace(user.Email)
		if email == "" {
			result.SkippedCount++
			continue
		}
		if err := sendAnnouncementBroadcastEmail(req.Title, email, req.Content); err != nil {
			result.FailedCount++
			common.SysLog(fmt.Sprintf("failed to send announcement email to user %d: %s", user.Id, err.Error()))
			continue
		}
		result.SentCount++
	}
	common.SysLog(fmt.Sprintf("announcement email broadcast source=%s target=%s sent=%d skipped=%d failed=%d", req.Source, req.Target, result.SentCount, result.SkippedCount, result.FailedCount))
	return result, nil
}

func listAnnouncementEmailRecipients(target string) ([]model.User, error) {
	query := model.DB.
		Select("id", "email", "role", "status", "user_type").
		Where("status = ?", common.UserStatusEnabled)

	switch target {
	case AnnouncementEmailTargetAgent:
		query = query.Where("COALESCE(user_type, '') = ?", model.UserTypeAgent)
	case AnnouncementEmailTargetEndUser:
		query = query.Where("role < ? AND COALESCE(user_type, '') <> ?", common.RoleAdminUser, model.UserTypeAgent)
	case AnnouncementEmailTargetAll:
		query = query.Where("role < ?", common.RoleAdminUser)
	default:
		return nil, fmt.Errorf("invalid target: %s", target)
	}

	var users []model.User
	if err := query.Order("id asc").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
```

- [ ] **Step 4: Run service tests to verify GREEN**

Run:

```powershell
go test ./service -run BroadcastAnnouncementEmail -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit backend service**

```powershell
git add service/announcement_email_broadcast.go service/announcement_email_broadcast_test.go
git commit -m "feat: add announcement email broadcast service"
```

## Task 2: Backend Controller And Route

**Files:**
- Create: `controller/announcement_email_broadcast.go`
- Modify: `router/api-router.go`
- Test manually through service tests and route compile.

- [ ] **Step 1: Write the route/controller compile target**

The controller will be covered by `go test ./controller ./router` compile checks. Create `controller/announcement_email_broadcast.go` with the expected handler shape during implementation, then add the route.

- [ ] **Step 2: Implement controller**

Create `controller/announcement_email_broadcast.go`:

```go
package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func BroadcastAnnouncementEmail(c *gin.Context) {
	var req service.AnnouncementEmailBroadcastRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}

	result, err := service.BroadcastAnnouncementEmail(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    result,
		})
		return
	}
	success := result.SentCount > 0 || result.FailedCount == 0
	message := ""
	if !success {
		message = "邮件发送失败"
	}
	c.JSON(http.StatusOK, gin.H{
		"success": success,
		"message": message,
		"data":    result,
	})
}
```

- [ ] **Step 3: Register route**

Modify `router/api-router.go` near `apiRouter.GET("/notice", controller.GetNotice)`:

```go
apiRouter.GET("/notice", controller.GetNotice)
apiRouter.POST("/notice/email-broadcast", middleware.RootAuth(), controller.BroadcastAnnouncementEmail)
```

- [ ] **Step 4: Run backend compile checks**

Run:

```powershell
go test ./controller ./router ./service -run BroadcastAnnouncementEmail -count=1
```

Expected: PASS or no tests in controller/router with successful compile.

- [ ] **Step 5: Commit backend API**

```powershell
git add controller/announcement_email_broadcast.go router/api-router.go
git commit -m "feat: expose announcement email broadcast api"
```

## Task 3: Shared Frontend Email Modal

**Files:**
- Create: `web/src/components/settings/AnnouncementEmailBroadcastModal.jsx`
- Create: `web/src/components/settings/AnnouncementEmailBroadcastModal.source.test.js`

- [ ] **Step 1: Write failing source test**

Create `web/src/components/settings/AnnouncementEmailBroadcastModal.source.test.js`:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const source = fs.readFileSync(
  new URL('./AnnouncementEmailBroadcastModal.jsx', import.meta.url),
  'utf8',
);

test('announcement email broadcast modal exposes editable target title and body', () => {
  assert.match(source, /Form\.Select[\s\S]*field='target'/);
  assert.match(source, /Form\.Input[\s\S]*field='title'/);
  assert.match(source, /Form\.TextArea[\s\S]*field='content'/);
});

test('announcement email broadcast modal renders edited content as email html before submit', () => {
  assert.match(source, /import\s+\{\s*marked\s*\}\s+from\s+'marked'/);
  assert.match(source, /marked\.parse\(values\.content\s*\|\|\s*''\)/);
  assert.match(source, /API\.post\('\/api\/notice\/email-broadcast'/);
});
```

- [ ] **Step 2: Run source test to verify RED**

Run:

```powershell
cd web
node --test src/components/settings/AnnouncementEmailBroadcastModal.source.test.js
```

Expected: FAIL because the component file does not exist.

- [ ] **Step 3: Implement modal component**

Create `web/src/components/settings/AnnouncementEmailBroadcastModal.jsx`:

```jsx
import React, { useEffect, useRef, useState } from 'react';
import { Button, Form, Modal, Space, Typography } from '@douyinfe/semi-ui';
import { Send } from 'lucide-react';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;

const targetOptions = (t) => [
  { label: t('代理商'), value: 'agent' },
  { label: t('普通用户'), value: 'end_user' },
  { label: t('全量用户'), value: 'all' },
];

const defaultStats = { sent_count: 0, skipped_count: 0, failed_count: 0 };

const AnnouncementEmailBroadcastModal = ({
  visible,
  source,
  defaultTitle,
  defaultContent,
  onClose,
}) => {
  const { t } = useTranslation();
  const formApiRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState(null);

  useEffect(() => {
    if (!visible) return;
    setStats(null);
    formApiRef.current?.setValues({
      target: 'all',
      title: defaultTitle || '',
      content: defaultContent || '',
    });
  }, [visible, defaultTitle, defaultContent]);

  const handleSend = async () => {
    const values = formApiRef.current?.getValues() || {};
    if (!values.target || !values.title?.trim() || !values.content?.trim()) {
      showError(t('请填写完整的邮件信息'));
      return;
    }

    setLoading(true);
    try {
      const res = await API.post('/api/notice/email-broadcast', {
        source,
        target: values.target,
        title: values.title.trim(),
        content: marked.parse(values.content || ''),
      });
      const { success, message, data } = res.data;
      const nextStats = data || defaultStats;
      setStats(nextStats);
      if (success) {
        showSuccess(
          `${t('邮件发送完成')}：${t('已发送')} ${nextStats.sent_count}，${t('跳过')} ${nextStats.skipped_count}，${t('失败')} ${nextStats.failed_count}`,
        );
        onClose?.();
      } else {
        showError(message || t('邮件发送失败'));
      }
    } catch (err) {
      showError(err.message || t('邮件发送失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title={t('邮件发送确认')}
      visible={visible}
      onCancel={onClose}
      footer={
        <Space>
          <Button onClick={onClose}>{t('取消')}</Button>
          <Button
            type='primary'
            icon={<Send size={14} />}
            loading={loading}
            onClick={handleSend}
          >
            {t('发送邮件')}
          </Button>
        </Space>
      }
      width={720}
    >
      <Form
        layout='vertical'
        getFormApi={(api) => {
          formApiRef.current = api;
        }}
        initValues={{
          target: 'all',
          title: defaultTitle || '',
          content: defaultContent || '',
        }}
      >
        <Form.Select
          field='target'
          label={t('接收用户')}
          optionList={targetOptions(t)}
          rules={[{ required: true, message: t('请选择接收用户') }]}
        />
        <Form.Input
          field='title'
          label={t('邮件标题')}
          rules={[{ required: true, message: t('请输入邮件标题') }]}
        />
        <Form.TextArea
          field='content'
          label={t('邮件正文')}
          autosize={{ minRows: 8, maxRows: 16 }}
          rules={[{ required: true, message: t('请输入邮件正文') }]}
        />
      </Form>
      {stats ? (
        <Text type='tertiary'>
          {`${t('已发送')} ${stats.sent_count}，${t('跳过')} ${stats.skipped_count}，${t('失败')} ${stats.failed_count}`}
        </Text>
      ) : null}
    </Modal>
  );
};

export default AnnouncementEmailBroadcastModal;
```

- [ ] **Step 4: Run source test to verify GREEN**

Run:

```powershell
cd web
node --test src/components/settings/AnnouncementEmailBroadcastModal.source.test.js
```

Expected: PASS.

- [ ] **Step 5: Commit modal**

```powershell
git add web/src/components/settings/AnnouncementEmailBroadcastModal.jsx web/src/components/settings/AnnouncementEmailBroadcastModal.source.test.js
git commit -m "feat: add announcement email broadcast modal"
```

## Task 4: Notice Save Integration

**Files:**
- Modify: `web/src/components/settings/OtherSetting.jsx`
- Create: `web/src/components/settings/OtherSetting.emailBroadcast.source.test.js`

- [ ] **Step 1: Write failing source test**

Create `web/src/components/settings/OtherSetting.emailBroadcast.source.test.js`:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const source = fs.readFileSync(new URL('./OtherSetting.jsx', import.meta.url), 'utf8');

test('other setting prompts optional email broadcast after notice save', () => {
  assert.match(source, /AnnouncementEmailBroadcastModal/);
  assert.match(source, /setNoticeEmailConfirmVisible\(true\)/);
  assert.match(source, /defaultTitle=\{t\('系统通知'\)\}/);
  assert.match(source, /defaultContent=\{noticeEmailDraft\}/);
});
```

- [ ] **Step 2: Run source test to verify RED**

Run:

```powershell
cd web
node --test src/components/settings/OtherSetting.emailBroadcast.source.test.js
```

Expected: FAIL because `OtherSetting.jsx` has no email broadcast integration.

- [ ] **Step 3: Implement notice integration**

Modify `web/src/components/settings/OtherSetting.jsx`:

Add import:

```jsx
import AnnouncementEmailBroadcastModal from './AnnouncementEmailBroadcastModal';
```

Add state near existing modal state:

```jsx
const [noticeEmailConfirmVisible, setNoticeEmailConfirmVisible] = useState(false);
const [noticeEmailModalVisible, setNoticeEmailModalVisible] = useState(false);
const [noticeEmailDraft, setNoticeEmailDraft] = useState('');
```

Update `submitNotice` success branch:

```jsx
await updateOption('Notice', inputs.Notice);
setNoticeEmailDraft(inputs.Notice || '');
showSuccess(t('公告已更新'));
setNoticeEmailConfirmVisible(true);
```

Render before the existing update modal:

```jsx
<Modal
  title={t('发送邮件确认')}
  visible={noticeEmailConfirmVisible}
  onOk={() => {
    setNoticeEmailConfirmVisible(false);
    setNoticeEmailModalVisible(true);
  }}
  onCancel={() => setNoticeEmailConfirmVisible(false)}
  okText={t('是')}
  cancelText={t('否')}
>
  <Text>{t('是否将此次通知以邮件形式发送？')}</Text>
</Modal>
<AnnouncementEmailBroadcastModal
  visible={noticeEmailModalVisible}
  source='notice'
  defaultTitle={t('系统通知')}
  defaultContent={noticeEmailDraft}
  onClose={() => setNoticeEmailModalVisible(false)}
/>
```

- [ ] **Step 4: Run source test to verify GREEN**

Run:

```powershell
cd web
node --test src/components/settings/OtherSetting.emailBroadcast.source.test.js
```

Expected: PASS.

- [ ] **Step 5: Commit notice integration**

```powershell
git add web/src/components/settings/OtherSetting.jsx web/src/components/settings/OtherSetting.emailBroadcast.source.test.js
git commit -m "feat: prompt notice email broadcast after save"
```

## Task 5: System Announcement Save Integration

**Files:**
- Modify: `web/src/pages/Setting/Dashboard/SettingsAnnouncements.jsx`
- Create: `web/src/pages/Setting/Dashboard/SettingsAnnouncements.emailBroadcast.source.test.js`

- [ ] **Step 1: Write failing source test**

Create `web/src/pages/Setting/Dashboard/SettingsAnnouncements.emailBroadcast.source.test.js`:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const source = fs.readFileSync(new URL('./SettingsAnnouncements.jsx', import.meta.url), 'utf8');

test('settings announcements tracks latest add or edit for optional email broadcast', () => {
  assert.match(source, /AnnouncementEmailBroadcastModal/);
  assert.match(source, /setLatestEmailAnnouncementDraft/);
  assert.match(source, /setAnnouncementEmailConfirmVisible\(true\)/);
  assert.match(source, /defaultTitle=\{t\('系统公告'\)\}/);
  assert.match(source, /source='announcement'/);
});

test('settings announcements delete paths clear email draft instead of prompting', () => {
  assert.match(source, /setLatestEmailAnnouncementDraft\(null\)/);
});
```

- [ ] **Step 2: Run source test to verify RED**

Run:

```powershell
cd web
node --test src/pages/Setting/Dashboard/SettingsAnnouncements.emailBroadcast.source.test.js
```

Expected: FAIL because system-announcement email integration is not present.

- [ ] **Step 3: Implement system announcement integration**

Modify `web/src/pages/Setting/Dashboard/SettingsAnnouncements.jsx`.

Add import:

```jsx
import AnnouncementEmailBroadcastModal from '../../../components/settings/AnnouncementEmailBroadcastModal';
```

Add state:

```jsx
const [announcementEmailConfirmVisible, setAnnouncementEmailConfirmVisible] = useState(false);
const [announcementEmailModalVisible, setAnnouncementEmailModalVisible] = useState(false);
const [latestEmailAnnouncementDraft, setLatestEmailAnnouncementDraft] = useState(null);
```

In `handleSaveAnnouncement`, after `formData` is built and before closing modal:

```jsx
setLatestEmailAnnouncementDraft(formData);
```

In `confirmDeleteAnnouncement` and `handleBatchDelete`, clear the draft:

```jsx
setLatestEmailAnnouncementDraft(null);
```

In `submitAnnouncements`, after `await updateOption('console_setting.announcements', announcementsJson);`:

```jsx
if (latestEmailAnnouncementDraft?.content) {
  setAnnouncementEmailConfirmVisible(true);
}
```

Render modals near existing modals:

```jsx
<Modal
  title={t('发送邮件确认')}
  visible={announcementEmailConfirmVisible}
  onOk={() => {
    setAnnouncementEmailConfirmVisible(false);
    setAnnouncementEmailModalVisible(true);
  }}
  onCancel={() => setAnnouncementEmailConfirmVisible(false)}
  okText={t('是')}
  cancelText={t('否')}
>
  <Text>{t('是否将此次系统公告以邮件形式发送？')}</Text>
</Modal>
<AnnouncementEmailBroadcastModal
  visible={announcementEmailModalVisible}
  source='announcement'
  defaultTitle={t('系统公告')}
  defaultContent={latestEmailAnnouncementDraft?.content || ''}
  onClose={() => setAnnouncementEmailModalVisible(false)}
/>
```

- [ ] **Step 4: Run source test to verify GREEN**

Run:

```powershell
cd web
node --test src/pages/Setting/Dashboard/SettingsAnnouncements.emailBroadcast.source.test.js
```

Expected: PASS.

- [ ] **Step 5: Commit system announcement integration**

```powershell
git add web/src/pages/Setting/Dashboard/SettingsAnnouncements.jsx web/src/pages/Setting/Dashboard/SettingsAnnouncements.emailBroadcast.source.test.js
git commit -m "feat: prompt announcement email broadcast after save"
```

## Task 6: Verification And Polish

**Files:**
- Modify locale files only if `bun run i18n:*` tooling requires synchronized keys.
- No new files expected unless tests reveal a missing source test.

- [ ] **Step 1: Run backend focused tests**

```powershell
go test ./service ./controller ./router -run "BroadcastAnnouncementEmail|TestStatus" -count=1
```

Expected: PASS.

- [ ] **Step 2: Run frontend source tests**

```powershell
cd web
node --test src/components/settings/AnnouncementEmailBroadcastModal.source.test.js src/components/settings/OtherSetting.emailBroadcast.source.test.js src/pages/Setting/Dashboard/SettingsAnnouncements.emailBroadcast.source.test.js
```

Expected: PASS.

- [ ] **Step 3: Run i18n extraction check if new keys need synchronization**

```powershell
cd web
bun run i18n:extract
bun run i18n:sync
```

Expected: commands complete without corrupting Chinese locale files. If they modify locale JSON files, inspect the exact diff for readable Chinese keys before committing.

- [ ] **Step 4: Run broader compile/build checks**

```powershell
go test ./...
cd web
bun run build
```

Expected: PASS. If full `go test ./...` is too slow or blocked by unrelated existing failures, record the exact failing package and rerun the focused packages.

- [ ] **Step 5: Final diff review**

```powershell
git diff --check
git status --short
```

Expected: no whitespace errors. Only files touched by this feature are staged or reported as changed by this feature; unrelated pre-existing changes remain untouched.

- [ ] **Step 6: Commit verification adjustments**

```powershell
git add service/announcement_email_broadcast.go service/announcement_email_broadcast_test.go controller/announcement_email_broadcast.go router/api-router.go web/src/components/settings/AnnouncementEmailBroadcastModal.jsx web/src/components/settings/AnnouncementEmailBroadcastModal.source.test.js web/src/components/settings/OtherSetting.jsx web/src/components/settings/OtherSetting.emailBroadcast.source.test.js web/src/pages/Setting/Dashboard/SettingsAnnouncements.jsx web/src/pages/Setting/Dashboard/SettingsAnnouncements.emailBroadcast.source.test.js
git commit -m "feat: broadcast saved announcements by email"
```

Skip this commit if all changed files were already committed in earlier tasks.

## Self-Review

- Spec coverage: backend targeting, editable title/body, optional confirmation, separate email draft, notice and announcement entry points, and verification are all mapped to tasks.
- Placeholder scan: no open placeholder steps remain.
- Type consistency: `source`, `target`, `title`, `content`, `sent_count`, `skipped_count`, and `failed_count` match between frontend, controller, service, and tests.
