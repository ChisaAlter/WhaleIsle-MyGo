package main

import (
	"encoding/json"
	"errors"
	"path"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
)

type officePreviewState struct {
	store   *session.Store
	session string
	path    string
	message string
}

func (s *mainState) setOfficePreviewMessage(sessions *session.Store, sid, path, message string) {
	s.mu.Lock()
	s.officePreview = officePreviewState{store: sessions, session: sid, path: path, message: message}
	s.mu.Unlock()
	sessions.SetLastAction(message)
}

func (s *mainState) officePreviewMessage(sessions *session.Store, sid, path string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.officePreview
	if status.store != sessions || status.session != sid || status.path != path {
		return ""
	}
	return status.message
}

func officePreviewName(source string) string {
	name := path.Base(strings.ReplaceAll(source, "\\", "/"))
	return strings.TrimSuffix(name, path.Ext(name)) + ".pdf"
}

func officePreviewReady(fonts []string) string {
	message := "Office 转换完成，PDF 只读预览已打开。"
	if len(fonts) > 0 {
		message += " 缺失字体：" + strings.Join(fonts, "、") + "。本次预览无法使用这些字体，文字和排版可能与原文档不同。"
	}
	return message
}

func officePreviewFailure(err error) string {
	unavailable := "Office 预览不可用。请在运行 Whale Isle Host 的主机上启用文档预览服务，并确认 LibreOffice Kit 运行依赖可用；然后点击“转为 PDF 预览”重试。"
	var remote *dsh.Error
	if errors.As(err, &remote) {
		switch remote.Code {
		case "gateway/invocation-unavailable", "gateway/service-unavailable":
			return unavailable
		case "gateway/cancelled":
			return "Office 预览已取消；点击“转为 PDF 预览”可重试。"
		case "document-render/failed":
			var details struct {
				Reason string `json:"reason"`
			}
			_ = json.Unmarshal(remote.Details, &details)
			switch details.Reason {
			case "input-too-large", "output-too-large":
				return "Office 文件或转换后的 PDF 超过预览大小上限；请缩小文件后重试。"
			case "invalid-document", "unsupported-format":
				return "无法预览此 Office 文件。文件可能已损坏、受密码保护，或与扩展名不符。"
			case "timeout":
				return "Office 转换超时；点击“转为 PDF 预览”可重试。"
			case "unavailable":
				return unavailable
			case "busy":
				return "Office 预览任务较多；请稍后点击“转为 PDF 预览”重试。"
			case "source-changed":
				return "Office 文件在读取时已更改；请重新打开预览。"
			default:
				return "Office 转换失败，未生成可用的 PDF；请检查文件后点击“转为 PDF 预览”重试。"
			}
		}
	}
	if strings.Contains(err.Error(), "response exceeds bounded wire limit") {
		return "转换后的 PDF 超过预览传输大小上限；请缩小文件后重试。"
	}
	if strings.Contains(err.Error(), "harness not ready") {
		return unavailable
	}
	return "Office 预览失败：" + err.Error() + "。点击“转为 PDF 预览”可重试。"
}
