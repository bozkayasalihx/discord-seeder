package session

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/m1guelpf/chatgpt-discord/src/ref"
	"github.com/playwright-community/playwright-go"
)

func GetSession() (string, error) {
	runOptions := playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	}
	err := playwright.Install(&runOptions)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Couldn't install headless browser: %v", err))
	}

	pw, err := playwright.Run(&runOptions)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Couldn't start headless browser: %v", err))
	}

	browser, page, err := launchBrowser(pw, "https://chat.openai.com", true)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Couldn't launch headless browser: %v", err))
	}

	for page.URL() != "https://chat.openai.com/chat" {
		result := <-logIn(pw)
		if result.Error != nil {
			return "", errors.New(fmt.Sprintf("Couldn't log in: %v", result.Error))
		}

		authCookie := playwright.BrowserContextAddCookiesOptionsCookies{
			Path:     ref.Of("/"),
			Secure:   ref.Of(true),
			HttpOnly: ref.Of(true),
			Value:    ref.Of(result.SessionToken),
			Domain:   ref.Of("chat.openai.com"),
			SameSite: playwright.SameSiteAttributeLax,
			Name:     ref.Of("__Secure-next-auth.session-token"),
			Expires:  ref.Of(float64(time.Now().AddDate(0, 1, 0).Unix())),
		}

		csrfcookie := playwright.BrowserContextAddCookiesOptionsCookies{
			Path:     ref.Of("/"),
			Secure:   ref.Of(true),
			HttpOnly: ref.Of(true),
			Value:    ref.Of("f3a34527a33d2b2a6b45755dd69fd1dae49813754810cb3db591398e88891bc5%7Cbf41c0c344eb7cb8bc382bc88a15aa174777822752bb268c4caeb8f356f965c5"),
			Domain:   ref.Of("chat.openai.com"),
			SameSite: playwright.SameSiteAttributeLax,
			Name:     ref.Of("__Host-next-auth.csrf-token"),
			Expires:  ref.Of(float64(time.Now().AddDate(0, 1, 0).Unix())),
		}

		if err := browser.AddCookies(authCookie, csrfcookie); err != nil {
			return "", errors.New(fmt.Sprintf("Couldn't save session to browser: %v", err))
		}

		if _, err = page.Goto("https://chat.openai.com/chat"); err != nil {
			return "", errors.New(fmt.Sprintf("Couldn't reload page: %v", err))
		}
	}

	sessionToken, err := getSessionCookie(browser)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Couldn't get session token: %v", err))
	}

	if err := browser.Close(); err != nil {
		return "", errors.New(fmt.Sprintf("Couldn't close headless browser: %v", err))
	}
	if err := pw.Stop(); err != nil {
		return "", errors.New(fmt.Sprintf("Couldn't stop headless browser: %v", err))
	}

	return sessionToken, nil
}

func launchBrowser(pw *playwright.Playwright, url string, headless bool) (playwright.BrowserContext, playwright.Page, error) {
	browser, err := pw.Chromium.LaunchPersistentContext("/tmp/chatgpt", playwright.BrowserTypeLaunchPersistentContextOptions{Headless: playwright.Bool(headless)})
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Couldn't launch headless browser: %v", err))
	}
	page, err := browser.NewPage()
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Couldn't create a new tab on headless browser: %v", err))
	}

	if _, err = page.Goto(url); err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Couldn't open website: %v", err))
	}

	return browser, page, nil
}

type Result struct {
	Error        error
	SessionToken string
}

// note: read all session information from the cookies;
func logIn(pw *playwright.Playwright) <-chan Result {
	var lock sync.Mutex
	r := make(chan Result)

	lock.Lock()
	go func() {
		defer close(r)
		defer lock.Unlock()

		browser, page, err := launchBrowser(pw, "https://chat.openai.com/", false)
		if err != nil {
			r <- Result{Error: errors.New(fmt.Sprintf("Couldn't launch headless browser: %v", err))}
			return
		}
		log.Println("Please log in to OpenAI Chat")

		page.On("framenavigated", func(frame playwright.Frame) {
			if frame.URL() != "https://chat.openai.com/chat" {
				return
			}

			lock.Unlock()
		})

		lock.Lock()

		sessionToken, err := getSessionCookie(browser)
		if err != nil {
			r <- Result{Error: errors.New(fmt.Sprintf("Couldn't get session token: %v", err))}
			return
		}

		if err := browser.Close(); err != nil {
			r <- Result{Error: errors.New(fmt.Sprintf("Couldn't close headless browser: %v", err))}
			return
		}

		r <- Result{SessionToken: sessionToken}
	}()

	return r
}

func getSessionCookie(browser playwright.BrowserContext) (string, error) {
	cookies, err := browser.Cookies("https://chat.openai.com/")
	if err != nil {
		return "", errors.New(fmt.Sprintf("Couldn't get cookies: %v", err))
	}

	var sessionToken string
	for _, cookie := range cookies {
		if cookie.Name == "__Secure-next-auth.session-token" {
			sessionToken = "eyJhbGciOiJkaXIiLCJlbmMiOiJBMjU2R0NNIn0..mUc8LfFS7cMLhjKJ.-mz-D8SEyEZTZAbTu73Tv2CcnFTg9THLGxeFHeK_h_hKIubOaHa5mGs2UQ-fjt4yxYqdI-BwZ3k9QjvMKDOl2e9Pdc--26n20QfmbOm3c9N0C0qLLh4IzmGKP4BM2FAmS5PuHMq1JdjPSe1VaoVQdo8K7R__VqkhCdHt5Gbzwonkv-8v4KluXcwfvLVlvROCDa69eMQXQ-v9IuOxQkBl2HZbQylzI-zbMzJjAXG7ik3WpCZdFrekq5o0QiRqlS8Mly2Emqb1nHW2-aJxF7i2SHAOVViYHKgBH6s-Jdo7nFC3UpqdNTfJBOJ6VwN3mLiX0Eg_fcJYTEuKGMgAwN8_GVl-T0g5jSHthMj4c0zS_i0hsMOq0r4aeiJz9L6iLkJi8TVg6nhfz9Wi7648fcIyLIzPZJ9mxxu_BZQPd2QT6tkjg6jqZL8dTKyvFimAEPXNHd0CV1lCwGShI5Vsg4SeCvgctW8oEzZH0G3DLyBBPLh6njyww4UP4bmiAUwaeChhk_pGFwPU4Y9IIGOA8KL6gLLQ6tcB2FwBZ3i9KMuc5dlWWoqb8IAF50XLucGbri1gTRScR61ibkd4CabIp1cYA_nn5o7w_3oVjibFhnVzmmnpADb6z-eeEpv_cbU6-wDx7gL99u_uEkBUB--_Q6NOUlwD-vyc0OsYilJ1uM9WXqqn0_XtPMbd-XKLKFxateAxTtFsuDu_BuRaLxe7uEKP6PkR3AzTTS0lukjsfunDMGCnCyi1aGzVnvJXxdF8aCt4ALMDs_ErEuiyAlO1Y6PvNbttOdtKvX6UdTZZKq4RAelMfCPTMX-W7GPP-0TYerfLK8m_re-wmyA0K7uaIIS_SIU_wVV0wCzwcLSd4OyGGgSmBEsD5CWeWVgCtzU3PxNfLsck4gCEzOEWZPVQI4-4-TEJlR7bZl5hRGyWSqDyywdoRqYv7XyEsEAYRK0IEdrTHKE5pknuN0Vgppti9r4EAwBNwMd7M5-yrk5mNtmTh-2YiHqeiqcuLNo4zPf2vwn2tYZ-suBUK3SW97HjGWrq8br1PIoJmaNPYYVYuYQgLyTSVGe-WmM9yCseHwZ-fq3W31X58E2WPco8h3UCq2fdy3ZQ2oUKy8NmtVcJ0_pi5toz9WY-2eR-k5IcEeR216YHoUjtfy7FAlb0tR7m1d6hmfRhAmjzyFc65GscYLLqBCvwe7KzPDzfeLTFYnJtzHYPRsFyTD8F-pq7eMzMt_dtM2-2ZLUF_ZhJo_uIsoud5XsZSJJiUPYdQIe8o_Sbc6AKFmCMFsqr5o9iOR2nS52M2M28SlqmkMrwexeI6lc5eRV0aSqZOc0dodMm1u0eNjhBnzlRyfVRn8hGu0L6Ju80OQVcpINzbu4Cf67UmFdYlfIm-hTbLFGAyUram-X3LJiQ8sOR6nH4ZgyZlYc1jHPrvBFHggZXGmGCQEZvi97tbjagiuIG3VjxY5oqXD9Sx5vucC99ACf7J9ckcYjKlr06zDlQ2HM8wst7_cB7MnmEYRsNDWf3GbJ3NUApbvLkWxYtcTgfvpZ2zoSG_c65d0yu_-KjWeG2omo66YcEEGx70sWM8BOJ7Ty28CL_q_zM2fLe4SEL9Scuf1obk2ahv9RSDYFKQ9LepwADUQnyIXAbUzUFXJ9uFHR-EXiMlxiJJKyzHKCgOTEIvBtLyQLJWWI5BH429_79kBI5ozDV9NtGXI-y-W0B87OACbum9VB4RwMtRYliXcFc0oFeVUmVwT5xo1fvb4QfuThM8NZHwwVdKzpI_bSBim5Ndk4xalloGOhaUILpx9TS0Vv9OCvRK7Su5nFZdF2R3E3_xKS9cmLZTaVgT2tc0l_G_4n7CXoWtRUwJpmJ4agsz1YWCDylQMN9Kr1sKc7YcDKeVbKqlBaseJ5YNDbUuH_gOT9GYX5DeIzK0hvReZebVL8DMavjCB9KRqkHmDmTMSxuPmLMIb8JdLBo0GFeoXR8GaTYKp0VsYcDKlYlWANxj79c9AcuXWUsQnZTmQ3YdhpPRJhXzvnD2e0fooZharMZvQwffZ4d_I-_ej7EVOAlJ2Ng3v330sbYgynD6B1aRz00Xkre2JI1w6Z0s99VVPh9Y1HuED8NUEjH8hKoTnDUuvUYoqRcxaeLYqVAXxJXISegIpr96glVEEZeaaNgJwK2HvUX6EIHAJK95In7yDLmnFaXK4xHV__SARe9VtevISFbH9E78dNawGAIDkabdc7SmswCLxWKhOyK_Z1ZDvireQNiA7hlyqDji8PQos-BBcAMf4Ntk44fUfPoaWn2YgYhhpZJzep6vp8QkOQHXJJ30yPwU4ALPlyGMBWo1r1gNnQzVFV_yPtEpFhepsPn30bNvxaNbcfFwnO1LsBvBM_E7h2Za5fngwD_RAQTf0Bt2U0NB2ldhQfbFb5sTEqzqsNGcO3POpKnGUKRVsUvUs_TZF883vFenVmoKSu5OL8JZNgIl9P0TCZaQ0uN7zN_r6Wxtbch1xDw-hnOE_Dsqwy8oFGNL1XwSASHl6a4dbTreh1nbpS5YgEvuobI4Q.5M-J4-nc0Cwj0IpKwWKYeg"
			break
		}
	}

	if sessionToken == "" {
		return "", errors.New(fmt.Sprintf("Couldn't get session token"))
	}

	return sessionToken, nil
}
