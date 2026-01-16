package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type ZkpLoginRequest struct {
	ZkpCode string `json:"zkpCode" binding:"required"`
	AffCode string `json:"affCode"`
}

// abbreviateAddress returns abbreviated wallet address like "0x1234...abcd"
func abbreviateAddress(address string) string {
	if len(address) <= 10 {
		return address
	}
	return address[:6] + "..." + address[len(address)-4:]
}

func ZkpOAuth(c *gin.Context) {
	// Check if ZKP private key is configured
	if common.ZkpPrivateKey == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "ZKP authentication is not configured",
		})
		return
	}

	var req ZkpLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "INVALID_PAYLOAD",
		})
		return
	}

	// Parse zkpCode
	payload, err := service.ParseZkpCode(req.ZkpCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "INVALID_ZKP_CODE",
		})
		return
	}

	// Verify proof and write to chain
	walletAddress, txHash, err := service.VerifyProof(payload)
	if err != nil {
		common.SysLog("ZKP verification failed: " + err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "PROOF_INVALID",
		})
		return
	}

	// Check club membership
	if !service.IsClubMember(walletAddress) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "NOT_CLUB_MEMBER",
		})
		return
	}

	// Get zkp hash from payload
	zkpHash := payload.Input[0].String()

	// Check if user exists
	user := model.User{
		WalletAddress: walletAddress,
	}

	if model.IsWalletAddressAlreadyTaken(walletAddress) {
		// User exists, fill user data
		err := user.FillUserByWalletAddress()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}

		// Check if user has been deleted
		if user.Id == 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "用户已注销",
			})
			return
		}

		// Update zkp hash
		user.ZkpHash = zkpHash
		err = user.Update(false)
		if err != nil {
			common.SysLog("Failed to update zkp hash: " + err.Error())
		}
	} else {
		// Create new user
		if !common.RegisterEnabled {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "管理员关闭了新用户注册",
			})
			return
		}

		user.Username = abbreviateAddress(walletAddress)
		user.DisplayName = abbreviateAddress(walletAddress)
		user.Role = common.RoleCommonUser
		user.Status = common.UserStatusEnabled
		user.ZkpHash = zkpHash
		user.Group = "vip"

		// Check for affiliation code from request
		inviterId := 0
		if req.AffCode != "" {
			inviterId, _ = model.GetUserIdByAffCode(req.AffCode)
		}

		if err := user.Insert(inviterId); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}

	// Check user status
	if user.Status != common.UserStatusEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户已被封禁",
		})
		return
	}

	// Setup login session
	setupZkpLogin(&user, txHash, c)
}

func setupZkpLogin(user *model.User, txHash string, c *gin.Context) {
	session := sessions.Default(c)
	session.Set("id", user.Id)
	session.Set("username", user.Username)
	session.Set("role", user.Role)
	session.Set("status", user.Status)
	session.Set("group", user.Group)
	err := session.Save()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "无法保存会话信息，请重试",
			"success": false,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "",
		"success": true,
		"data": map[string]any{
			"id":             user.Id,
			"username":       user.Username,
			"display_name":   user.DisplayName,
			"role":           user.Role,
			"status":         user.Status,
			"group":          user.Group,
			"wallet_address": user.WalletAddress,
			"tx_hash":        txHash,
		},
	})
}
