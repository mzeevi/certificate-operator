package certhandler

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_Decoder(t *testing.T) {
	type args struct {
		data     string
		password string
	}
	type want struct {
		tlsData TLSData
		err     error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ShouldDecodeDataSuccessfully": {
			args: args{
				data:     "MIIKKQIBAzCCCeUGCSqGSIb3DQEHAaCCCdYEggnSMIIJzjCCBg8GCSqGSIb3DQEHAaCCBgAEggX8MIIF+DCCBfQGCyqGSIb3DQEMCgECoIIE/jCCBPowHAYKKoZIhvcNAQwBAzAOBAi/wGZzoSMKIwICB9AEggTYxFtxHGzOCroXq6x/oX7qxJMB9y9NbAGcqBYg6ItIG01SZQd8UacOuHIZTdvmOOhwTDG/lU+Z+bPMnaxGnj6i2i2ePgS616rXQGy5IN2IpgJQWDHBYrHYXO7F6dipRQoe2/HSgV3rZFWkIy5qXmnshHS63VY7HFgTxmSA+fpNqU5apCcGCLqAnxTAl4gjlsIRDutawZsh10HTotYZs4Et6UuVukvvOf0BnuU6eKIatirj4cdOm8odS09+cpc/uakY16Elx6/yTCZFUAOU/qlFRmilt3CwogbX7wza2QkAyXhwY8G95ijHOZYeeIofQFJtR0JKyzzmKXP++oV94BqZTvVQoDG0iW6JFtCJrU4kovg19rs9hIUTbwdo7znoKtKQtMFeD1En78L/XiWQtnpfKVRk6IYCr55amCKYXFDogl6ntSr2TAJd3qQIH0vLD+/7Y52ZBEinuHUnMNtqUDQUrUJlliNTPtmSeYicvIaiDsUEyawZPU2uD5k086dPYd7pZhpqmYK6z7mw476AyDnvCgLcY1+L8lyTXrxKHa+zHFKjP+fK/PDZCdHItgobJPp63Cuv3+2qc1gWdTkcxDUVGvyLCTiZQGXWVPI8AKuGjqxsCg/xueYSYkgrU2vtd793eN2rsZlivWzoeGgiironVjbmMqsftcKFghZLNvvrUaJl/I0NW52Puwh+HvnwsQYie5PlP9H3uNpDEjGhX4nF7or7cCOFdnZLZIBfnRs/X7RYOeVipon9EozX1NbzxjdpoMvplfP57ydLLFFaN8fi6B8cyvksDKb0pFmwMTW8QzsckGXEGi8ap6iikxIsaT0j3iDkINt1IdiPfAxwYnQylmAYsVkmp+HWeaQdX1xq2BICxLXGqian1FznOghvNToS8zeS0BzMdTXspYAOojXCpxWZD/rWL2lD7X3Jkf4kVVl4w0tTcjInhB/N0dZ7wYiq7UqtvnaMHQDlkg3SW+XDlCZNo6RINtpafZxarSNj44RoPGQX1Ajxa/YtXGLrocNeRw43p3Vt93kg7mOCW0jSYsoFdzuZcNypYxU4ks2n7azn6utfR/FGcyifHthlyETfZRx+H6s3fLrc9TYyXUtm0JbApKcIEvf3F0oOuyXnELzb0Td2IurtQCo3v619TrwYaffPrDhSkgCxLkiExpoytQMdP8XdnggOFApt3CFmZxrz2veg+HoIO0f9PGPLwyzm5jWOrZx2Yrczi3vD4EV5Z+Um4S/0m7jQPolFyGO8FiSSHS1Kpv9UE7lWVvTzbyn5a7CHlw787DbDNSC+Pph7TGId/6I9z2x+5TXYx68KepCX24FLXQgpJO+GEaLK5mf1J97OAIUIYH5pwn5xAU3URtknZmiF2AKF4dEuQ2/1H0m4hawZ9rsidVx6YNQpPQhDZ8gAcdmtep36Pw0lVT6InucKxRkxH5n8OtR/66eD/K5BQzHBuieQnUGoDjuvAQ0G6gx9AXrJixjeosfF6jpp/o+NPOw83AlJXGABhORCj5pPkZmhqauo+4LUjs9kPvu3FJp2h7DFE3LUgm4mzi2n8qJdDhRqf6OWHuDcYcvgwo9rMHOxG8g9Vl5jwiCG0VxbHg8OmNoUITPjSIZyHQLF6XX9A3QP0qD72PGxyPrZHAdhW/8jOA7PoTGB4jANBgkrBgEEAYI3EQIxADATBgkqhkiG9w0BCRUxBgQEAQAAADBdBgkqhkiG9w0BCRQxUB5OAHQAZQAtADEAMgBmADcANgAzADcAYgAtADEAZQA1AGMALQA0AGQANwBhAC0AOQA3AGYANAAtAGEAYwBkAGQAZAA4AGUAZgBhADIANAAzMF0GCSsGAQQBgjcRATFQHk4ATQBpAGMAcgBvAHMAbwBmAHQAIABTAHQAcgBvAG4AZwAgAEMAcgB5AHAAdABvAGcAcgBhAHAAaABpAGMAIABQAHIAbwB2AGkAZABlAHIwggO3BgkqhkiG9w0BBwagggOoMIIDpAIBADCCA50GCSqGSIb3DQEHATAcBgoqhkiG9w0BDAEDMA4ECHTc2zCDnIFPAgIH0ICCA3DBpSRq62GTlcR9qY50s2hAwPVoUPzbuYfysucRTOQL5/K+SufWV9dYe8HDSrLdjcbDzZh1AaC5szXx6JoKb+k3EZvO4ijzPnbq0bXXeTynWqF5Qy940gKXYcD9bZIBzzAGTw5bAMkVHNWz6aLG0eXiPeoYt8edXpAwWqVEKpGNicC1uC6aayqhKbEyQXG7tqLgmexll86IsBw8jNJfhOc4hkVZoDriu7riwSmPXEyJ0/PKNDUujemnzSLkcto7TqAhWuVpuDu8/SkvVAT94Pboc62h88NaTPSnAdu6TWpiqYJUksURi+9jBJigpJGhGTYwZ870hAw650L28xTdHfcf67RItDnkAjXvGcySVcNq7OAshQ/8D3jE7jxX/wL/bzOTnM1D0tm+O5E8QuYGdYdovgUFpfwGwZT2bLwhKKsNKPW03H3EsqnSlEPtoAVecOC/ePp30E9JYJGzwinavLGryu/rl5dpQ7du5CqiufM2VsrT0N12Bv3GCFbyscX3wh8VSgmYYloH4gYkwqetw4m7Mth1cyas0gmbxyJDNLjzCqIwF6mhc12aZjfwwFqizDMhZqjiQU88jaFKBYBWxSrXiDdUzp/IBZQDoL4Ja8Qu6lPbg9RGZEh2nmsK8L2qD0cR92SGh9RobzVDIlOBOSBdypncZuogvukedL7SpfVcooFmQvlvWgxwNXb4Hk7yBtAq8E87eNjDlaYABJx6qG6QRXw0Dl6m9YZjCUqjF7Sm8738iKeYVQVwTOSEBeYQg73H7ZykyXOQ/KZqX+tOnXWOx1/JeNl1h+//W87+oiGlap9346kbODObGlRQKXg2huN2a3/a0pRQx9Ma/o/th6MpdIgD8xA0dtWovWZTEn/wL1bYA68UZIvLjCgqgvFaM7tYGJyGNsuD1qU/++yTxFGINN556tBQqOE1Pahic/k23zhXGrhQkBDkvl9Vpr3kyH0of2zxxfxr8kwjgzWnPbi8kxRYt/rUtAMAE1RWIwdmthb/j6JOoelWng9GA2wguJ5K8TFU+0hfhHc1tpLNJndRuhTNJSzfSTnuSvn2k+agmEJ59Z9DWSb4ODmG/1leT/PpW9FNkTS3M2NpgAxWQgNYJ+hIxBpOMBkSr8Dy+vS86DqboLmtDFmewCzycBuZeeEg+uWpfU/B1zGGrPVhFAeIMDswHzAHBgUrDgMCGgQUmD/myrmnzxzk9ni3ZWlVcvh0E58EFENUGqxY3LZ66Gosv4mVtJYzUGqTAgIH0A==",
				password: "jtvdDUG0E7Ll",
			},
			want: want{
				tlsData: TLSData{
					CertificateBytes: []uint8(`-----BEGIN CERTIFICATE-----`),
					PrivateKeyBytes:  []uint8(`-----BEGIN RSA PRIVATE KEY-----`),
				},
				err: nil,
			},
		},
		"ShouldFailToDecodeData": {
			args: args{
				data:     "wrong-data",
				password: "wrong-password",
			},
			want: want{
				tlsData: TLSData{},
				err:     fmt.Errorf(errCannotDecodeB64Data, "illegal base64 data at input byte 5"),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tlsData, err := Decoder(tc.args.data, tc.args.password)
			if !bytes.Contains(tlsData.CertificateBytes, tc.want.tlsData.CertificateBytes) {
				t.Fatalf("Decoder(...): expected certificate bytes not found in result")
			}

			if !bytes.Contains(tlsData.PrivateKeyBytes, tc.want.tlsData.PrivateKeyBytes) {
				t.Fatalf("Decoder(...): expected private key bytes not found in result")
			}

			if err != nil {
				if diff := cmp.Diff(tc.want.err.Error(), err.Error()); diff != "" {
					t.Fatalf("Decoder(...): -want error, +got error: %v", diff)
				}
			}
		})
	}
}
