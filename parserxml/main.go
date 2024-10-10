package main

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// var errorMessages = map[string]string{
// 	"E001": "病歷號無健檢紀錄",
// 	"E002": "資料解析失敗",
// 	"E003": "XML 檔案讀取失敗",
// 	"E005": "XML 格式解析失敗",
// 	"E006": "GetDataResult JSON 解析失敗",
// 	"E007": "病歷號無健檢紀錄",
// 	"E010": "defusedcolname 必須包含至少一個欄位名稱",
// 	"E011": "欄位不存在",
// 	"E012": "解析欄位時發生錯誤",
// 	"E999": "未知的錯誤",
// }

var operations = []struct {
	parseFunc func(string, []string) (interface{}, error) // 解析函式
	colLabel  string                                      // 解析欄位說明
	colName   string                                      // 解析後要存入 parsedJSON 的欄位名稱
	fields    []string                                    // 欄位名稱陣列
}{
	{calculateAge, "年齡", "Age", []string{"基本資料-出生日期", "基本資料-健檢日期"}},
	{parseSex, "性別", "Sex", []string{"基本資料-性別"}},
	{parseDiseaseHistory, "高血壓", "HT", []string{"過去疾病史-高血壓V3", "Medicine : 使用藥物-降血壓藥物V3"}},
	{parseDiseaseHistory, "冠狀動脈疾病", "CAD", []string{"過去疾病史-冠狀動脈疾病V3"}},
	{parseDiseaseHistory, "心律不整", "Arrhythmia", []string{"過去疾病史-心律不整V3"}},
	{parseDiseaseHistory, "心辮膜疾病", "VHD", []string{"過去疾病史-心辮膜疾病V3"}},
	{parseDiseaseHistory, "腦中風", "Stroke", []string{"過去疾病史-腦中風V3"}},
	{parseDiseaseHistory, "糖尿病", "DM", []string{"過去疾病史-糖尿病V3", "Medicine : 使用藥物-降血糖藥物V3", "Medicine : 使用藥物-施打胰島素V3"}},
	{parseDiseaseHistory, "高血脂症", "Hyperlipidemia", []string{"過去疾病史-高血脂症V3", "Medicine : 使用藥物-降血脂藥物V3"}},
	{parseDiseaseHistory, "高尿酸症或痛風", "Gout", []string{"過去疾病史-高尿酸症或痛風V3"}},
	{parseDiseaseHistory, "甲狀腺機能亢進", "Hyperthyroidism", []string{"過去疾病史-甲狀腺機能亢進V3"}},
	{parseDiseaseHistory, "甲狀腺機能低下", "Hypothyroidism", []string{"過去疾病史-甲狀腺機能低下V3"}},
	{parseDiseaseHistory, "膽囊或膽管結石", "gallstones", []string{"過去疾病史-膽囊或膽管結石V3"}},
	{parseDiseaseHistory, "大腸息肉", "colon_polyp", []string{"過去疾病史-大腸息肉V3"}},
	{parseDiseaseHistory, "痔瘡", "hemorrhoid", []string{"過去疾病史-痔瘡V3"}},
	{parseDiseaseHistory, "慢性胰臟炎", "chronic_pancreatitis", []string{"過去疾病史-慢性胰臟炎V3"}},
	{parseDiseaseHistory, "風濕性關節炎", "RA", []string{"過去疾病史-風濕性關節炎V3"}},
	{parseDiseaseHistory, "腎炎", "Nephritis", []string{"過去疾病史-腎炎V3"}},
	{parseDiseaseHistory, "腎病症候群及病變", "Nephrotic_syndrome", []string{"過去疾病史-腎病症候群及病變V3"}},
	{parseDiseaseHistory, "急性腎衰竭", "AKI", []string{"過去疾病史-急性腎衰竭V3"}},
	{parseDiseaseHistory, "慢性腎衰竭", "CKD", []string{"過去疾病史-慢性腎衰竭V3"}},
	{parseDiseaseHistory, "腎結石或尿路結石", "nephrolithiasis", []string{"過去疾病史-腎結石或尿路結石V3"}},
	{parseDiseaseHistory, "泌尿道感染", "UTI", []string{"過去疾病史-泌尿道感染V3"}},
	{parseDiseaseHistory, "攝護腺肥大", "Prostate_hypertrophy", []string{"過去疾病史-攝護腺肥大V3"}},
	{parseDiseaseHistory, "青光眼", "glaucoma", []string{"過去疾病史-青光眼V3"}},
	{parseDiseaseHistory, "白內障", "cataract", []string{"過去疾病史-白內障V3"}},
	{parseDiseaseHistory, "高度近視", "nearsighted", []string{"過去疾病史-高度近視V3"}},
	{parseDiseaseHistory, "視網膜病變", "Retinopathy", []string{"過去疾病史-視網膜病變V3"}},
	{parseDiseaseHistory, "中耳炎", "Otitis_media", []string{"過去疾病史-中耳炎V3"}},
	{parseDiseaseHistory, "梅尼爾氏症-內耳神經失調", "Menieres_disease", []string{"過去疾病史-梅尼爾氏症-內耳神經失調V3"}},
	{parseDiseaseHistory, "肺炎", "pneumonia", []string{"過去疾病史-肺炎V3"}},
	{parseDiseaseHistory, "肺氣腫", "Emphysema", []string{"過去疾病史-肺氣腫V3"}},
	{parseDiseaseHistory, "肺結核", "TB", []string{"過去疾病史-肺結核V3"}},
	{parseDiseaseHistory, "支氣管擴張症", "Bronchiectasis", []string{"過去疾病史-支氣管擴張症V3"}},
	{parseDiseaseHistory, "慢性支氣管炎", "chronic_bronchitis", []string{"過去疾病史-慢性支氣管炎V3"}},
	{parseDiseaseHistory, "氣喘", "asthma", []string{"過去疾病史-氣喘V3"}},
	{parseDiseaseHistory, "紅斑性狼瘡", "SLE", []string{"過去疾病史-紅斑性狼瘡V3"}},
	{parseDiseaseHistory, "逆流性食道炎", "GERD", []string{"過去疾病史-逆流性食道炎V3"}},
	{parseDiseaseHistory, "胃或十二指腸潰瘍", "PUD", []string{"過去疾病史-胃或十二指腸潰瘍V3"}},
	{parseDiseaseHistory, "B型肝炎帶原", "hepatitis_B", []string{"過去疾病史-B型肝炎帶原V3"}},
	{parseDiseaseHistory, "C型肝炎帶原", "hepatitis_C", []string{"過去疾病史-C型肝炎帶原V3"}},
	{parseDiseaseHistory, "肝硬化", "Liver cirrhosis", []string{"過去疾病史-肝硬化V3"}},
	{parseDiseaseHistory, "急性胰臟炎", "Acute_Pancreatitis", []string{"過去疾病史-急性胰臟炎V3"}},
	{parseDiseaseHistory, "聽力障礙", "Hearing_impairments", []string{"過去疾病史-聽力障礙V3"}},
	{parseDiseaseHistory, "鼻中隔彎曲", "NSD", []string{"過去疾病史-鼻中隔彎曲V3"}},
	{parseDiseaseHistory, "過敏性鼻炎", "Allergic_rhinitis", []string{"過去疾病史-過敏性鼻炎V3"}},
	{parseDiseaseHistory, "鼻竇炎", "Sinusitis", []string{"過去疾病史-鼻竇炎V3"}},
	{parseDiseaseHistory, "癲癇", "epilepsy", []string{"過去疾病史-癲癇V3"}},
	{parseDiseaseHistory, "巴金森氏症", "Parkinson", []string{"過去疾病史-巴金森氏症V3"}},
	{parseDiseaseHistory, "躁鬱症", "bipolar_disorder", []string{"過去疾病史-躁鬱症V3"}},
	{parseDiseaseHistory, "憂鬱症", "Depression", []string{"過去疾病史-憂鬱症V3"}},
	{parseDiseaseHistory, "睡眠呼吸中止症候群", "sleep_apnea", []string{"過去疾病史-睡眠呼吸中止症候群V3"}},
	{parseDiseaseHistory, "癌症", "cancer", []string{"Status : 身體狀況-鼻咽癌V3", "Status : 身體狀況-口腔癌/下咽癌V3", "Status : 身體狀況-肺癌V3", "Status : 身體狀況-乳癌V3", "Status : 身體狀況-食道癌V3", "Status : 身體狀況-胃癌V3", "Status : 身體狀況-肝癌V3", "Status : 身體狀況-胰臟癌V3", "Status : 身體狀況-膽囊癌/膽管癌V3", "Status : 身體狀況-大腸直腸癌V3", "Status : 身體狀況-卵巢癌V3", "Status : 身體狀況-子宮頸癌V3", "Status : 身體狀況-攝護腺癌V3", "Status : 身體狀況-白血病V3", "Status : 身體狀況-淋巴癌V3", "Status : 身體狀況-其他惡性腫瘤V3"}},
	{parseDiseaseHistory, "降血糖藥物", "OHA", []string{"Medicine : 使用藥物-降血糖藥物V3"}},
	{parseDiseaseHistory, "施打胰島素", "insulin", []string{"Medicine : 使用藥物-施打胰島素V3"}},
	{parseDiseaseHistory, "降血壓藥物", "antihypertensive", []string{"Medicine : 使用藥物-降血壓藥物V3"}},
	{parseDiseaseHistory, "利尿劑", "diuretic", []string{"Medicine : 使用藥物-利尿劑V3"}},
	{parseDiseaseHistory, "降血脂藥物", "lipid_lowering", []string{"Medicine : 使用藥物-降血脂藥物V3"}},
	{parseDiseaseHistory, "抗凝血劑", "Anticoagulant", []string{"Medicine : 使用藥物-抗凝血劑V3"}},
	{parseDiseaseHistory, "抗血小板藥", "Antiplatelet", []string{"Medicine : 使用藥物-抗血小板藥V3"}},
	{parseDiseaseHistory, "心律不整藥", "Antiarrhythmic", []string{"Medicine : 使用藥物-心律不整藥V3"}},
	{parseDiseaseHistory, "降尿酸藥物", "uricacid_drug", []string{"Medicine : 使用藥物-降尿酸藥物V3"}},
	{parseDiseaseHistory, "氣喘肺氣腫", "asthma_drug", []string{"Medicine : 使用藥物-氣喘肺氣腫V3"}},
	{parseDiseaseHistory, "感冒藥", "cold_drug", []string{"Medicine : 使用藥物-感冒藥V3"}},
	{parseDiseaseHistory, "胃藥物", "Stomach_drug", []string{"Medicine : 使用藥物-胃藥物V3"}},
	{parseDiseaseHistory, "肝臟病藥物", "liver_drug", []string{"Medicine : 使用藥物-肝臟病藥物V3"}},
	{parseDiseaseHistory, "甲狀腺藥物", "thyroid_drug", []string{"Medicine : 使用藥物-甲狀腺藥物V3"}},
	{parseDiseaseHistory, "止痛消炎藥", "anti_inflammatory", []string{"Medicine : 使用藥物-止痛消炎藥V3"}},
	{parseDiseaseHistory, "安眠鎮定劑", "Hypnotic", []string{"Medicine : 使用藥物-安眠鎮定劑V3"}},
	{parseDiseaseHistory, "荷爾蒙補充", "Hormone_sup", []string{"Medicine : 使用藥物-荷爾蒙補充V3"}},
	{parseDiseaseHistory, "骨質疏鬆藥", "osteoporosis_drug", []string{"Medicine : 使用藥物-骨質疏鬆藥V3"}},
	{parseDiseaseHistory, "鐵劑", "iron_sup", []string{"Medicine : 使用藥物-鐵劑V3"}},
	{parseDiseaseHistory, "類固醇", "Steroid", []string{"Medicine : 使用藥物-類固醇V3"}},
	{parseDiseaseHistory, "攝護腺藥物", "Prostate_drug", []string{"Medicine : 使用藥物-攝護腺藥物V3"}},
	{parseFamilyHistory, "高血壓家族史", "fami_HT", []string{"Family History : 家族病史-高血壓V3"}},
	{parseFamilyHistory, "冠狀動脈疾病(心絞痛或心肌梗塞)家族史", "fami_CAD", []string{"Family History : 家族病史-冠狀動脈疾病(心絞痛或心肌梗塞)V3"}},
	{parseFamilyHistory, "腦中風家族史", "fami_Stroke", []string{"Family History : 家族病史-腦中風V3"}},
	{parseFamilyHistory, "糖尿病家族史", "fami_DM", []string{"Family History : 家族病史-糖尿病V3"}},
	{parseFamilyHistory, "高血脂症家族史", "fami_Hyperlipidemia", []string{"Family History : 家族病史-高血脂症V3"}},
	{parseFamilyHistory, "高尿酸症或痛風家族史", "fami_Gout", []string{"Family History : 家族病史-高尿酸症或痛風V3"}},
	{parseFamilyHistory, "逆流性食道炎家族史", "fami_GERD", []string{"Family History : 家族病史-逆流性食道炎V3"}},
	{parseFamilyHistory, "消化性潰瘍(胃潰瘍或十二指腸潰瘍)家族史", "fami_PUD", []string{"Family History : 家族病史-消化性潰瘍(胃潰瘍或十二指腸潰瘍)V3"}},
	{parseFamilyHistory, "B型肝炎帶原家族史", "fami_hepatitis_B", []string{"Family History : 家族病史-B型肝炎帶原V3"}},
	{parseFamilyHistory, "C型肝炎帶原家族史", "fami_hepatitis_C", []string{"Family History : 家族病史-C型肝炎帶原V3"}},
	{parseFamilyHistory, "肝硬化家族史", "fami_Liver_cirrhosis", []string{"Family History : 家族病史-肝硬化V3"}},
	{parseFamilyHistory, "慢性胰臟炎家族史", "fami_pancreatitis", []string{"Family History : 家族病史-慢性胰臟炎V3"}},
	{parseFamilyHistory, "膽囊或膽管結石家族史", "fami_gallstones", []string{"Family History : 家族病史-膽囊或膽管結石V3"}},
	{parseFamilyHistory, "肺氣腫家族史", "fami_Emphysema", []string{"Family History : 家族病史-肺氣腫V3"}},
	{parseFamilyHistory, "氣喘家族史", "fami_asthma", []string{"Family History : 家族病史-氣喘V3"}},
	{parseFamilyHistory, "紅斑性狼瘡家族史", "fami_SLE", []string{"Family History : 家族病史-紅斑性狼瘡V3"}},
	{parseFamilyHistory, "風濕性關節炎家族史", "fami_RA", []string{"Family History : 家族病史-風濕性關節炎V3"}},
	{parseFamilyHistory, "癌症家族史", "fami_cancer", []string{"Family History : 家族病史-鼻咽癌V3", "Family History : 家族病史-口腔癌/下咽癌V3", "Family History : 家族病史-肺癌V3", "Family History : 家族病史-乳癌V3", "Family History : 家族病史-食道癌V3", "Family History : 家族病史-胃癌V3", "Family History : 家族病史-肝癌V3", "Family History : 家族病史-胰臟癌V3", "Family History : 家族病史-膽管癌V3", "Family History : 家族病史-大腸直腸癌V3", "Family History : 家族病史-卵巢癌V3", "Family History : 家族病史-子宮頸癌V3", "Family History : 家族病史-攝護腺癌V3", "Family History : 家族病史-白血病V3", "Family History : 家族病史-淋巴癌V3", "Family History : 家族病史-其他惡性腫瘤V3"}},
	{parseBMI, "身體質量指數", "BMI", []string{"身體理學檢查-Body Height", "身體理學檢查-Body Weight", "身體理學檢查-BMI"}},
	{parseGirth, "腹圍", "Abd_Girth", []string{"身體理學檢查-Abd_Girth"}},
	{parseBP, "收縮壓", "SBP", []string{"身體理學檢查-Systolic Pressure(R)", "身體理學檢查-Systolic Pressure(L)"}},
	{parseBP, "舒張壓", "DBP", []string{"身體理學檢查-Diastolic Pressure(R)", "身體理學檢查-Diastolic Pressure(L)"}},
	{parsePass, "脂肪重", "MBF", []string{"身體理學檢查-MBF"}},
	{parsePass, "體脂肪百分比", "PBF", []string{"身體理學檢查-PBF"}},
	{parsePass, "基礎代謝率", "BMR", []string{"身體理學檢查-BMR"}},
	{parsePass, "阻抗係數", "IMP", []string{"身體理學檢查-IMP"}},
	{parsePass, "每日消耗總能量", "calory", []string{"身體理學檢查-calory"}},
	{parsePass, "內臟脂肪程度", "whr_level", []string{"身體理學檢查-whr_level"}},
	{parsePass, "內臟脂肪面積", "vfa", []string{"身體理學檢查-vfa"}},
	{parsePass, "內臟脂肪重量", "mvf_quantity", []string{"身體理學檢查-mvf_quantity"}},
	{parsePass, "皮下脂肪重量", "msf_quantity", []string{"身體理學檢查-msf_quantity"}},
	{parsePass, "細胞內水", "ICF", []string{"身體理學檢查-ICF"}},
	{parsePass, "細胞外水", "ECF", []string{"身體理學檢查-ECF"}},
	{parsePass, "水腫指數", "edema", []string{"身體理學檢查-edema"}},
	{parsePass, "肥胖度", "Fatness", []string{"身體理學檢查-Fatness"}},
	{parseAlcohol, "飲酒狀態", "alcohol_gp", []string{"Hobit : 生活習慣-飲酒頻率V3", "Hobit : 生活習慣-每週飲酒天數酒精濃度 ＜10%", "Hobit : 生活習慣-每週飲酒天數酒精濃度 11~20%", "Hobit : 生活習慣-每週飲酒天數酒精濃度 21~30%", "Hobit : 生活習慣-每週飲酒天數酒精濃度 31~40%", "Hobit : 生活習慣-每週飲酒天數酒精濃度 41~50%", "Hobit : 生活習慣-每週飲酒天數酒精濃度 51~60%", "Hobit : 生活習慣-每週飲酒天數酒精濃度 ＞60%"}},
	{parseCigarette, "吸菸狀態", "smoke_gp", []string{"Hobit : 生活習慣-現在平均每日抽菸量", "Hobit : 生活習慣-抽菸菸齡", "Hobit : 生活習慣-若您已戒菸了，已戒幾年", "Hobit : 生活習慣-戒菸前之平均每日抽菸量"}},
	{parseBetel, "檳榔嚼食", "betel_gp", []string{"Hobit : 生活習慣-每日嚼食檳榔量"}},
	{parseDrink, "咖啡習慣", "coffee_gp", []string{"Hobit : 生活習慣-平均每日喝咖啡量"}},
	{parseDrink, "喝茶習慣", "tea_gp", []string{"Hobit : 生活習慣-平均每日喝茶量"}},
	{parseFood, "飲食狀態", "food_gp", []string{"Hobit : 生活習慣-飲食種類"}},
	{parseExcercise, "運動狀態", "excercise_week", []string{"Hobit : 生活習慣-費力活動", "Hobit : 生活習慣-費力活動花多少時間", "Hobit : 生活習慣-中等費力活動", "Hobit : 生活習慣-中等費力活動花多少時間"}},
	{parsePSQI, "匹茲堡睡眠量表分數", "PSQI", []string{"Hobit : 生活習慣-過去一個月來，您每天睡眠的時間大約幾小時", "Hobit : 生活習慣-過去一個月來，您在上床後通常多久才能入睡", "Hobit : 生活習慣-過去一個月來，您實際每晚睡著時間佔躺床總時間的比例(即睡眠效率)約為多少", "Hobit : 生活習慣-過去一個月來，您的睡眠每星期有幾次出現下列困擾情形", "Hobit : 生活習慣-過去一個月來，整體而言，您覺得自己的睡眠品質如何", "Hobit : 生活習慣-過去一個月來，您有幾次需要使用藥物幫忙睡眠", "Hobit : 生活習慣-過去一個月來，您每星期約幾次曾在用餐、開車或社交場合瞌睡而無法保持清醒，或感到無心完成該做的事"}},
	{parseBSRS5, "簡式健康量表", "BSRS5", []string{"Hobit : 生活習慣-睡眠困難，譬如難以入睡、易醒或早醒", "Hobit : 生活習慣-感覺緊張不安", "Hobit : 生活習慣-覺得容易苦惱或動怒", "Hobit : 生活習慣-感覺憂鬱、心情低落", "Hobit : 生活習慣-覺得比不上別人"}},
	{parseZeroDiscard, "Albumin", "Albumin", []string{"血液及實驗室常規檢查-Albumin-ALB"}},
	{parseZeroDiscard, "ALP", "ALP", []string{"血液及實驗室常規檢查-ALP"}},
	{parseZeroDiscard, "ALT", "ALT", []string{"血液及實驗室常規檢查-ALT"}},
	{parseZeroDiscard, "AST", "AST", []string{"血液及實驗室常規檢查-AST"}},
	{parseZeroDiscard, "Creatinine", "Creatinine", []string{"血液及實驗室常規檢查-CRE"}},
	{parseZeroDiscard, "r_GT", "r_GT", []string{"血液及實驗室常規檢查-r-GT"}},
	{parseZeroDiscard, "Glucose_PC", "Glucose_PC", []string{"血液及實驗室常規檢查-GLU PC"}},
	{parseZeroDiscard, "Glucose_AC", "Glucose_AC", []string{"血液及實驗室常規檢查-GLU AC"}},
	{parseGlucoseU, "Glucose_U", "Glucose_U", []string{"血液及實驗室常規檢查-Glucose (U)"}},
	{parseZeroDiscard, "Hemoglobin", "Hemoglobin", []string{"血液及實驗室常規檢查-HB"}},
	{parseZeroDiscard, "HbA1c", "HbA1c", []string{"血液及實驗室常規檢查-HbA1c"}},
	{parseZeroDiscard, "HCT", "HCT", []string{"血液及實驗室常規檢查-HCT"}},
	{parseZeroDiscard, "HDL", "HDL", []string{"血液及實驗室常規檢查-HDL-C"}},
	{parseHsCRP, "hsCRP", "hsCRP", []string{"血液及實驗室常規檢查-hsCRP", "血液及實驗室常規檢查-hs-CRP 20160618停用"}},
	{parseZeroDiscard, "LDL", "LDL", []string{"血液及實驗室常規檢查-LDL-C"}},
	{parseZeroDiscard, "Platelet", "Platelet", []string{"血液及實驗室常規檢查-Platelet"}},
	{parseZeroDiscard, "T_BIL", "T_BIL", []string{"血液及實驗室常規檢查-T-BIL"}},
	{parseZeroDiscard, "Total_Cholesterol", "Total_Cholesterol", []string{"血液及實驗室常規檢查-T-CHO"}},
	{parseZeroDiscard, "Triglycerides", "Triglycerides", []string{"血液及實驗室常規檢查-TG"}},
	{parseZeroDiscard, "TP", "TP", []string{"血液及實驗室常規檢查-TP"}},
	{parseZeroDiscard, "Uric_acid", "Uric_acid", []string{"血液及實驗室常規檢查-UA"}},
	{parseZeroDiscard, "BUN", "BUN", []string{"血液及實驗室常規檢查-BUN"}},
	{parseZeroDiscard, "WBC", "WBC", []string{"血液及實驗室常規檢查-WBC"}},
	{parseZeroDiscard, "RBC", "RBC", []string{"血液及實驗室常規檢查-RBC"}},
	{parseAgatston, "冠狀動脈鈣化指數", "Agatston_score", []string{"PACS_相關影像檢查-Chest CT",
		"PACS_相關影像檢查-MDCT",
		"PACS_相關影像檢查-低輻射64切電腦斷層冠狀動脈血管攝影",
		"PACS_相關影像檢查-Low dose Coronary CT Angiography",
		"PACS_相關影像檢查-calcium score",
		"PACS_相關影像檢查-心臟鈣化指數"}},
}

// 定義錯誤碼回傳格式
type Result struct {
	Error       string                 `json:"error"`
	ErrorCode   string                 `json:"error_code"`
	ErrorDetail string                 `json:"error_detail"`
	Data        map[string]interface{} `json:"data"`
}

// func setError(result *Result, errorCode string, errorDetail string) {
// 	result.ErrorCode = errorCode
// 	result.Error = errorMessages[errorCode]
// 	result.ErrorDetail = errorDetail
// }

// 初始化 Result 結構
func initResult() Result {
	return Result{
		Error:       "",
		ErrorCode:   "",
		ErrorDetail: "",
		Data:        make(map[string]interface{}),
	}
}

// 定義對應 XML 結構的 Go 結構體
type Envelope struct {
	Body Body `xml:"Body"`
}

type Body struct {
	GetDataResponse GetDataResponse `xml:"GetDataResponse"`
}

type GetDataResponse struct {
	GetDataResult string `xml:"GetDataResult"`
}

// 定義回傳的 JSON 結構
type GetDataResultJSON struct {
	HaveData string            `json:"HaveData"`
	Data     map[string]string `json:"Data"`
}

// 讀取 XML 檔案
func readXmlFile(filePath string) (xmlData Envelope, result Result) {
	result = initResult()

	// 打開檔案
	file, err := os.Open(filePath)
	if err != nil {
		result.Error = "XML 檔案讀取失敗"
		result.ErrorCode = "E003"
		result.ErrorDetail = err.Error()
		return xmlData, result
	}
	defer file.Close()

	// 使用 io 套件讀取檔案內容
	var byteValue []byte
	buffer := make([]byte, 1024)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			result.Error = "XML 檔案讀取失敗"
			result.ErrorCode = "E003"
			result.ErrorDetail = err.Error()
			return xmlData, result
		}
		if n == 0 {
			break
		}
		byteValue = append(byteValue, buffer[:n]...)
	}

	// 解析 XML
	err = xml.Unmarshal(byteValue, &xmlData)
	if err != nil {
		result.Error = "XML 格式解析失敗"
		result.ErrorCode = "E005"
		result.ErrorDetail = err.Error()
	}
	return xmlData, result
}

func readTxtFile(path string) (dataJSON map[string]string, result Result) {
	// Initialize result
	result = initResult()

	// Read the file content
	fileContent, err := os.ReadFile(path)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to read TXT file: %v", err)
		return nil, result
	}

	// Unmarshal the JSON content into a map
	err = json.Unmarshal(fileContent, &dataJSON)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to parse JSON content: %v", err)
		return nil, result
	}

	// Successfully parsed JSON into a map
	return dataJSON, result
}

// 讀取文字轉成分鐘
func str2Min(timeStr string) (int, error) {
	// 移除前後空白
	timeStr = strings.TrimSpace(timeStr)
	if timeStr == "" {
		return 0, nil
	}

	// 嘗試直接將數字字串轉換為分鐘
	if num, err := strconv.Atoi(timeStr); err == nil {
		// 如果轉換成功，直接返回該數字作為分鐘數
		return num, nil
	}

	// 定義正則表達式來匹配
	hourMinutePattern := regexp.MustCompile(`(\d+)小時(\d+)?分鐘?`)
	hourPattern := regexp.MustCompile(`(\d+)小時`)
	minutePattern := regexp.MustCompile(`(\d+)分鐘`)

	// 處理 "X小時Y分鐘"
	if matches := hourMinutePattern.FindStringSubmatch(timeStr); matches != nil {
		hours, _ := strconv.Atoi(matches[1])
		minutes := 0
		if matches[2] != "" {
			minutes, _ = strconv.Atoi(matches[2])
		}
		return (hours * 60) + minutes, nil // 將小時和分鐘轉換為總分鐘數
	}

	// 處理 "X小時"
	if matches := hourPattern.FindStringSubmatch(timeStr); matches != nil {
		hours, _ := strconv.Atoi(matches[1])
		return hours * 60, nil // 只轉換小時為分鐘
	}

	// 處理 "Y分鐘"
	if matches := minutePattern.FindStringSubmatch(timeStr); matches != nil {
		minutes, _ := strconv.Atoi(matches[1])
		return minutes, nil // 直接返回分鐘數
	}

	// 如果無法匹配任何格式，返回錯誤
	return 0, fmt.Errorf("無法解析時間字串(分鐘): %s", timeStr)
}

// 讀取文字轉成天數
func str2Day(timeStr string) (int, error) {
	// 移除前後空白
	timeStr = strings.TrimSpace(timeStr)
	if timeStr == "" {
		return 0, nil
	}

	if num, err := strconv.Atoi(timeStr); err == nil {
		// 如果轉換成功，直接返回該數字作為天數
		return num, nil
	}

	// 定義正則表達式來匹配 "N天"
	DayPattern := regexp.MustCompile(`(\d+)天`)

	// 處理 "X小時Y分鐘"
	if matches := DayPattern.FindStringSubmatch(timeStr); matches != nil {
		Days, _ := strconv.Atoi(matches[1])
		return Days, nil // 將小時和分鐘轉換為總分鐘數
	}

	return 0, fmt.Errorf("無法解析時間字串(天): %s", timeStr)
}

// 轉換問題的回答為對應的分數
func convertToScore(response string, categories []string, scores []int) (int, error) {
	for i, category := range categories {
		if strings.Contains(response, category) {
			return scores[i], nil
		}
	}
	return 0, fmt.Errorf("無法解析回應: %s", response)
}

func extractGetDataResult(xmlData Envelope) (dataResult GetDataResultJSON, result Result) {
	result = initResult()

	// 取得 GetDataResult，這是一個 JSON 字串
	getDataResult := xmlData.Body.GetDataResponse.GetDataResult

	// 解析 GetDataResult 中的 JSON
	err := json.Unmarshal([]byte(getDataResult), &dataResult)
	if err != nil {
		result.Error = "GetDataResult JSON 解析失敗"
		result.ErrorCode = "E006"
		result.ErrorDetail = err.Error()
	}
	return dataResult, result
}

func checkHaveData(dataResult GetDataResultJSON) (dataJSON map[string]string, result Result) {
	dataJSON = dataResult.Data
	result = initResult()

	// 如果 HaveData 是 "no"，則返回錯誤
	if dataResult.HaveData == "no" {
		result.Error = "病歷號無健檢紀錄"
		result.ErrorCode = "E007"
	}
	return dataJSON, result
}

func extractdata(dataJSON map[string]string) (result Result) {
	parsedJSON := make(map[string]interface{})
	result = initResult()

	// result = parsedata(calculateAge, &parsedJSON, dataJSON, "Age", []string{"基本資料-出生日期", "基本資料-健檢日期"})
	// if result.Error != "" {
	// 	return result
	// }
	// result = parsedata(parseSex, &parsedJSON, dataJSON, "Sex", []string{"基本資料-性別"})
	// if result.Error != "" {
	// 	return result
	// }

	// 定義要解析的欄位及其對應的函式和參數

	// 使用迴圈來處理所有的解析
	for _, op := range operations {
		result = parsedata(op.parseFunc, &parsedJSON, dataJSON, op.colLabel, op.colName, op.fields)
		if result.Error != "" {
			return result
		}
	}

	result.Data = parsedJSON
	return result
}

func parsedata(parsefunction func(string, []string) (interface{}, error), parsedJSON *map[string]interface{}, data map[string]string, modelcollabel string, modelcolname string, defusedcolname []string) (result Result) {
	result = initResult()

	// 檢查 defusedcolname 是否至少包含一個欄位
	if len(defusedcolname) < 1 {
		result.Error = fmt.Sprintf("%s 必須包含至少一個欄位名稱", modelcollabel)
		result.ErrorCode = "E010"
		return result
	}

	// 提取所有指定的欄位值
	var fieldValues []string
	for _, colName := range defusedcolname {
		if val, exists := data[colName]; exists {
			fieldValues = append(fieldValues, val)
		} else {
			result.Error = fmt.Sprintf("原始資料欄位 %s 不存在", colName)
			result.ErrorCode = "E011"
			return result
		}
	}

	// 使用提供的 parsefunction 解析
	parsedValue, err := parsefunction(modelcollabel, fieldValues)
	if err != nil {
		result.Error = fmt.Sprintf("解析欄位 %s 時發生錯誤", modelcolname)
		result.ErrorCode = "E012"
		result.ErrorDetail = err.Error()
		return result
	}

	// 成功解析，將結果存入指向的 parsedJSON
	(*parsedJSON)[modelcolname] = parsedValue

	return result
}

func calculateAge(label string, fields []string) (age interface{}, err error) {
	var numFields int = 2
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	birthdate := strings.TrimSpace(fields[0])
	testdate := strings.TrimSpace(fields[1])
	layout := "2006/01/02"
	birth, err := time.Parse(layout, birthdate)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】的出生日期無法解析: %s", label, birthdate)
	}
	checkup, err := time.Parse(layout, testdate)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】的填表日期無法解析: %s", label, testdate)
	}

	age = int(checkup.Sub(birth).Hours() / 24 / 365.25)
	return age, nil
}

func parseSex(label string, fields []string) (interface{}, error) {
	var numFields int = 1
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	gender := strings.TrimSpace(fields[0])
	if gender == "男" {
		return 1, nil
	} else if gender == "女" {
		return 0, nil
	} else {
		return nil, fmt.Errorf("輸出欄位【%s】無法解析: %s", label, gender)
	}
}

func parseDiseaseHistory(label string, fields []string) (interface{}, error) {
	// 定義需要檢查的關鍵詞
	keywords := []string{"有", "治療", "藥物", "手術"}

	// 檢查 fields 中是否包含任何關鍵詞
	for _, field := range fields {
		for _, keyword := range keywords {
			if strings.Contains(field, keyword) {
				return 1, nil // 如果找到關鍵詞，返回 1
			}
		}
	}
	return 0, nil // 如果沒有找到，返回 0
}

func parseFamilyHistory(label string, fields []string) (interface{}, error) {
	// 定義需要檢查的關鍵詞
	keywords := []string{"父親", "母親", "祖父母", "外組父母"}

	// 檢查 fields 中是否包含任何關鍵詞
	for _, field := range fields {
		for _, keyword := range keywords {
			if strings.Contains(field, keyword) {
				return 1, nil // 如果找到關鍵詞，返回 1
			}
		}
	}
	return 0, nil // 如果沒有找到，返回 0
}

func parseBMI(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 3
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	// 解析身高、體重和輸入的 BMI
	heightStr := strings.TrimSpace(fields[0])
	weightStr := strings.TrimSpace(fields[1])
	bmiStr := strings.TrimSpace(fields[2])

	bmi, err := strconv.ParseFloat(bmiStr, 64)
	if err != nil {
		bmi = -1 // 當 BMI 是空白或無效時，設定為 -1 以表示需要重新計算
	}

	// 判斷 BMI 是否在合理範圍內 10 < BMI <= 60
	if bmi > 10 && bmi <= 60 {
		return bmi, nil // 直接使用解析出的 BMI
	}

	// 將字串轉換為浮點數
	height, err := strconv.ParseFloat(heightStr, 64)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】的身高數據無效: %s", label, heightStr)
	}

	weight, err := strconv.ParseFloat(weightStr, 64)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】的體重數據無效: %s", label, weightStr)
	}

	// 如果 BMI 超出範圍或無效，使用身高和體重計算
	if height > 0 && weight > 0 {
		// 計算 BMI
		calculatedBMI := weight / (height / 100) / (height / 100)

		// 確保計算出的 BMI 也在合理範圍內
		if calculatedBMI > 10 && calculatedBMI <= 60 {
			return calculatedBMI, nil
		}
	}

	// 如果無法得到有效的 BMI，返回錯誤
	return nil, fmt.Errorf("輸出欄位【%s】無法計算有效值", label)
}

func parseGirth(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 1
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	girthStr := strings.TrimSpace(fields[0])
	girth, err := strconv.ParseFloat(girthStr, 64)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】數據無效: %s", label, girthStr)
	}

	// 判斷腰圍是否合理
	if girth > 30 && girth <= 200 {
		return girth, nil // 直接原值
	} else {
		return nil, fmt.Errorf("輸出欄位【%s】數據不採用: %f", label, girth)
	}
}

func parseBP(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 2
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	// 解析右側和左側的血壓
	rightBPStr := strings.TrimSpace(fields[0])
	leftBPStr := strings.TrimSpace(fields[1])

	// 將字串轉換為浮點數，如果為空白則設為 0
	rightBP, err := strconv.ParseFloat(rightBPStr, 64)
	if err != nil || rightBPStr == "" {
		rightBP = 0
	}

	leftBP, err := strconv.ParseFloat(leftBPStr, 64)
	if err != nil || leftBPStr == "" {
		leftBP = 0
	}

	// 規則 1: 若左右血壓皆不為 0 或空白，則取平均值
	if rightBP > 0 && leftBP > 0 {
		averageBP := (rightBP + leftBP) / 2
		return averageBP, nil
	}

	// 規則 2: 若任一側為 0 或空白，使用另一側的數值
	if rightBP > 0 {
		return rightBP, nil
	}

	if leftBP > 0 {
		return leftBP, nil
	}

	// 規則 3: 若兩者皆為 0 或空白，則不採用，返回錯誤
	return nil, fmt.Errorf("無有效的%s數據", label)
}

func parsePass(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 1
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	valueStr := strings.TrimSpace(fields[0])

	// 將字串轉換為浮點數
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】數據無效，值: %s", label, valueStr)
	} else {
		return value, nil
	}
}

func parseAlcohol(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 8
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	// 解析欄位值
	frequency := strings.TrimSpace(fields[0])
	daysOfWeek := fields[1:]
	for key := range daysOfWeek {
		daysOfWeek[key] = strings.TrimSpace(daysOfWeek[key])
	}

	// 檢查天數欄位是否全部填答為 0 或空白
	allZero := (frequency != "0") && (frequency != "")
	for _, day := range daysOfWeek {
		if (day != "0") && (day != "") {
			allZero = false
			break
		}
	}

	// 根據規則進行解析
	if !allZero {
		return 2, nil // 每週天數並非填答 0 或空白
	} else if strings.Contains(frequency, "社交飲酒") {
		return 1, nil // 社交飲酒且每週天數皆為0
	} else if strings.Contains(frequency, "無飲酒") {
		return 0, nil // 無飲酒且每週天數皆為0
	}

	return nil, fmt.Errorf("輸出欄位【%s】數據無效", label)
}

func parseCigarette(label string, fields []string) (interface{}, error) {
	var numFields int = 4
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	dailyAmountStr := strings.TrimSpace(fields[0])
	yearsOfSmokeStr := strings.TrimSpace(fields[1])
	yearsOfQuitStr := strings.TrimSpace(fields[2])
	dailyAmountHistoryStr := strings.TrimSpace(fields[3])

	notSmokeKeyword := "不抽"
	emptyKeyword := ""

	// 情境 2：現在每日抽菸量 or 抽菸菸齡 任一非「不抽」或非空白
	// 情境 1：確認戒菸的情境
	// 情境 0：所有欄位皆為「不抽」或空白
	if (dailyAmountStr != notSmokeKeyword && dailyAmountStr != emptyKeyword) ||
		(yearsOfSmokeStr != notSmokeKeyword && yearsOfSmokeStr != emptyKeyword) {
		return 2, nil
	} else if (yearsOfQuitStr != notSmokeKeyword && yearsOfQuitStr != emptyKeyword) ||
		(dailyAmountHistoryStr != notSmokeKeyword && dailyAmountHistoryStr != emptyKeyword) {
		return 1, nil
	} else if (dailyAmountStr == notSmokeKeyword || dailyAmountStr == emptyKeyword) &&
		(yearsOfSmokeStr == notSmokeKeyword || yearsOfSmokeStr == emptyKeyword) &&
		(yearsOfQuitStr == notSmokeKeyword || yearsOfQuitStr == emptyKeyword) &&
		(dailyAmountHistoryStr == notSmokeKeyword || dailyAmountHistoryStr == emptyKeyword) {
		return 0, nil
	}

	// 若沒有匹配到任何情境，返回錯誤
	return nil, fmt.Errorf("輸出欄位【%s】數據無效", label)
}

// parseBetel 用來解析嚼檳榔的數據
func parseBetel(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 1
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	// 解析欄位值
	doItOrNotStr := strings.TrimSpace(fields[0])

	// 關鍵詞
	notChewKeyword := "不嚼"
	emptyKeyword := ""

	// 判斷邏輯
	if doItOrNotStr != notChewKeyword && doItOrNotStr != emptyKeyword {
		// 填答非空白且非「不嚼」
		return 1, nil
	} else if doItOrNotStr == notChewKeyword || doItOrNotStr == emptyKeyword {
		// 填答空白或是「不嚼」
		return 0, nil
	}

	// 若資料無法匹配任何情境，回傳錯誤
	return nil, fmt.Errorf("輸出欄位【%s】數據無效", label)
}

func parseDrink(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 1
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	// 解析欄位值
	doItOrNotStr := fields[0]

	// 關鍵詞
	notChewKeyword := "不喝"
	emptyKeyword := ""

	// 判斷邏輯
	if doItOrNotStr != notChewKeyword && doItOrNotStr != emptyKeyword {
		// 填答非空白且非「不喝」
		return 1, nil
	} else if doItOrNotStr == notChewKeyword || doItOrNotStr == emptyKeyword {
		// 填答空白或是「不喝」
		return 0, nil
	}

	return nil, fmt.Errorf("輸出欄位【%s】數據無效", label)
}

func parseFood(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 1
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	// 解析欄位值
	eatingStyle := fields[0]
	// 關鍵詞
	vegetarian := "素"

	if strings.Contains(eatingStyle, vegetarian) {
		return 0, nil // 如果找到關鍵詞，返回 0
	} else {
		return 1, nil
	}
}

func parseExcercise(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 4
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	// 解析欄位值並移除空白
	daysOfExcerciseStr := strings.TrimSpace(fields[0])
	dueTimeStr := strings.TrimSpace(fields[1])
	daysOfExcerciseMediumStr := strings.TrimSpace(fields[2])
	dueTimeMediumStr := strings.TrimSpace(fields[3])

	// 將字串轉換為數字
	daysOfExcercise, err := str2Day(daysOfExcerciseStr)
	if err != nil {
		return nil, fmt.Errorf("費力活動天數【%s】無法轉換為數字: %v", daysOfExcerciseStr, err)
	}

	dueTime, err := str2Min(dueTimeStr)
	if err != nil {
		return nil, fmt.Errorf("費力活動時間【%s】無法轉換為數字: %v", dueTimeStr, err)
	}

	daysOfExcerciseMedium, err := str2Day(daysOfExcerciseMediumStr)
	if err != nil {
		return nil, fmt.Errorf("中等費力活動天數【%s】無法轉換為數字: %v", daysOfExcerciseMediumStr, err)
	}

	dueTimeMedium, err := str2Min(dueTimeMediumStr)
	if err != nil {
		return nil, fmt.Errorf("中等費力活動時間【%s】無法轉換為數字: %v", dueTimeMediumStr, err)
	}

	// 計算總運動量
	totalExcerciseTime := (daysOfExcercise * dueTime * 2) + (daysOfExcerciseMedium * dueTimeMedium)

	// 返回計算結果
	return totalExcerciseTime, nil
}

// parsePSQI 解析 PSQI 問卷分數
func parsePSQI(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 7
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	totalScore := 0

	// 1. 睡眠時間 (題目 1)
	sleepTimeCategories := []string{"少於5小時", "5-6小時", "6-7小時", "7小時以上"}
	sleepTimeScores := []int{3, 2, 1, 0}
	score, err := convertToScore(fields[0], sleepTimeCategories, sleepTimeScores)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】第1題錯誤: %v", label, err)
	}
	totalScore += score

	// 2. 入睡時間 (題目 2)
	sleepLatencyCategories := []string{"61分鐘以上", "31-60分鐘", "15-30分鐘", "少於15分鐘"}
	sleepLatencyScores := []int{3, 2, 1, 0}
	score, err = convertToScore(fields[1], sleepLatencyCategories, sleepLatencyScores)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】第2題錯誤: %v", label, err)
	}
	totalScore += score

	// 3. 睡眠效率 (題目 3)
	sleepEfficiencyCategories := []string{"少於65%", "65-74%", "75-84%", "85%以上"}
	sleepEfficiencyScores := []int{3, 2, 1, 0}
	score, err = convertToScore(fields[2], sleepEfficiencyCategories, sleepEfficiencyScores)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】第3題錯誤: %v", label, err)
	}
	totalScore += score

	// 4. 睡眠困擾次數 (題目 4)
	sleepDisturbanceCategories := []string{"每週3次以上", "每週1-2次", "每週少於一次", "從未如此"}
	sleepDisturbanceScores := []int{3, 2, 1, 0}
	score, err = convertToScore(fields[3], sleepDisturbanceCategories, sleepDisturbanceScores)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】第4題錯誤: %v", label, err)
	}
	totalScore += score

	// 5. 睡眠品質 (題目 5)
	sleepQualityCategories := []string{"非常不好", "不好", "好", "非常好"}
	sleepQualityScores := []int{3, 2, 1, 0}
	score, err = convertToScore(fields[4], sleepQualityCategories, sleepQualityScores)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】第5題錯誤: %v", label, err)
	}
	totalScore += score

	// 6. 使用睡眠藥物 (題目 6)
	sleepMedicationCategories := []string{"每週3次以上", "每週1-2次", "每週少於一次", "從未如此"}
	sleepMedicationScores := []int{3, 2, 1, 0}
	score, err = convertToScore(fields[5], sleepMedicationCategories, sleepMedicationScores)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】第6題錯誤: %v", label, err)
	}
	totalScore += score

	// 7. 瞌睡情況 (題目 7)
	daytimeDysfunctionCategories := []string{"每週3次以上", "每週1-2次", "每週少於一次", "從未如此"}
	daytimeDysfunctionScores := []int{3, 2, 1, 0}
	score, err = convertToScore(fields[6], daytimeDysfunctionCategories, daytimeDysfunctionScores)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】第7題錯誤: %v", label, err)
	}
	totalScore += score

	// 回傳解析結果
	return totalScore, nil
}

func parseBSRS5(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 5
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	totalScore := 0

	// 1. 睡眠困難 (題目 1)
	sleepDifficultyCategories := []string{"非常厲害", "厲害", "中等", "輕微", "沒有"}
	sleepDifficultyScores := []int{4, 3, 2, 1, 0}
	score, err := convertToScore(fields[0], sleepDifficultyCategories, sleepDifficultyScores)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】第1題錯誤: %v", label, err)
	}
	totalScore += score

	// 2. 緊張不安 (題目 2)
	nervousnessCategories := []string{"非常厲害", "厲害", "中等", "輕微", "沒有"}
	nervousnessScores := []int{4, 3, 2, 1, 0}
	score, err = convertToScore(fields[1], nervousnessCategories, nervousnessScores)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】第2題錯誤: %v", label, err)
	}
	totalScore += score

	// 3. 容易苦惱或動怒 (題目 3)
	angerCategories := []string{"非常厲害", "厲害", "中等", "輕微", "沒有"}
	angerScores := []int{4, 3, 2, 1, 0}
	score, err = convertToScore(fields[2], angerCategories, angerScores)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】第3題錯誤: %v", label, err)
	}
	totalScore += score

	// 4. 憂鬱、心情低落 (題目 4)
	depressionCategories := []string{"非常厲害", "厲害", "中等", "輕微", "沒有"}
	depressionScores := []int{4, 3, 2, 1, 0}
	score, err = convertToScore(fields[3], depressionCategories, depressionScores)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】第4題錯誤: %v", label, err)
	}
	totalScore += score

	// 5. 覺得比不上別人 (題目 5)
	inferiorityCategories := []string{"非常厲害", "厲害", "中等", "輕微", "沒有"}
	inferiorityScores := []int{4, 3, 2, 1, 0}
	score, err = convertToScore(fields[4], inferiorityCategories, inferiorityScores)
	if err != nil {
		return nil, fmt.Errorf("輸出欄位【%s】第5題錯誤: %v", label, err)
	}
	totalScore += score

	// 回傳解析結果
	return totalScore, nil
}

func parseZeroDiscard(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 1
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個輸入欄位來解析輸出欄位【%s】，而輸入為%d個", numFields, label, len(fields))
	}

	// 解析欄位值，移除空白並嘗試轉換為浮點數
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return nil, fmt.Errorf("欄位【%s】中的數據無法轉換為浮點數: %s", label, fields[0])
	}

	// 檢查是否為0，若為0則不採用
	if value == 0 {
		return nil, fmt.Errorf("欄位【%s】數據為0，未採用", label)
	}

	// 若數值非0，直接回傳數值
	return value, nil
}

func parseGlucoseU(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 1
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個欄位來解析%s，而輸入為%d個", numFields, label, len(fields))
	}

	// 解析欄位值
	glucoseValue := fields[0]

	// 根據規則進行判斷
	switch {
	case strings.Contains(glucoseValue, "2+") || strings.Contains(glucoseValue, "3+") || strings.Contains(glucoseValue, "4+"):
		return 3, nil
	case strings.Contains(glucoseValue, "1+"):
		return 2, nil
	case strings.Contains(glucoseValue, "+/-") || strings.Contains(glucoseValue, "+-"):
		return 1, nil
	case strings.Contains(glucoseValue, "normal") || strings.Contains(glucoseValue, "NORMAL") || glucoseValue == "0" || glucoseValue == "-":
		return 0, nil
	}

	// 若資料無法匹配任何情境，回傳錯誤
	return nil, fmt.Errorf("輸出欄位【%s】數據無效", label)
}

func parseHsCRP(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 2
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個欄位來解析%s，而輸入為%d個", numFields, label, len(fields))
	}

	// 解析欄位值
	hsCRPValue := strings.TrimSpace(fields[0])
	hsCRPDeprecatedValue := strings.TrimSpace(fields[1])

	// 檢查 hsCRP 是否有效
	if hsCRPValue != "" && hsCRPValue != "0" {
		// 處理特殊情況
		switch hsCRPValue {
		case "<0.01", "&lt;0.01":
			return 0.005, nil
		case "<0.02", "&lt;0.02":
			return 0.01, nil
		default:
			floatValue, err := strconv.ParseFloat(hsCRPValue, 64)
			if err != nil {
				return 0, fmt.Errorf("輸出欄位【%s】第1個輸入無法轉換為浮點數: %v", label, err)
			}
			return floatValue, nil
		}
	}

	// 若 hsCRP 為空值或 0，檢查 hs-CRP 20160618 停用的值
	if hsCRPDeprecatedValue != "" && hsCRPDeprecatedValue != "0" {
		// 處理特殊情況
		switch hsCRPDeprecatedValue {
		case "<0.01":
			return 0.005, nil
		case "<0.02":
			return 0.01, nil
		default:
			floatValue, err := strconv.ParseFloat(hsCRPValue, 64)
			if err != nil {
				return 0, fmt.Errorf("輸出欄位【%s】第2個輸入無法轉換為浮點數: %v", label, err)
			}
			return floatValue, nil
		}
	}

	// 若兩個值均為空或 0，返回錯誤
	return nil, fmt.Errorf("輸出欄位【%s】數據無效", label)
}

func parseAgatston(label string, fields []string) (interface{}, error) {
	// 檢查輸入的欄位數量
	var numFields int = 6
	if len(fields) != numFields {
		return nil, fmt.Errorf("需要確切%d個欄位來解析%s，而輸入為%d個", numFields, label, len(fields))
	}

	// 提取 Agatston 分數
	agatstonRegex := regexp.MustCompile(`(?:Agatston score:|心臟冠狀動脈總鈣化積分:|Total:|calcium score:|total score:|coronary artery analysis:)\s*([0-9]+(?:\.[0-9]+)?)`)
	arteryRegex := regexp.MustCompile(`\*\s*(LM|LAD|LCX|RCA):\s*([0-9]+(?:\.[0-9]+)?)`)

	// 初始化變數以儲存分數
	var agatstonScore = make(map[string]float64)
	var arteryScores = make(map[string]float64)

	// 檢查每個欄位以提取Agatston分數和動脈分數
	for _, field := range fields {
		// 查找Agatston分數
		// if matches := agatstonRegex.FindStringSubmatch(field); len(matches) > 1 {
		// 	score, err := strconv.ParseFloat(matches[1], 64)
		// 	if err == nil {
		// 		agatstonScore = score
		// 	}
		// }

		for _, line := range strings.Split(field, ",") {
			if matches := agatstonRegex.FindStringSubmatch(line); len(matches) > 0 {
				agatston := matches[0]
				score, _ := strconv.ParseFloat(matches[1], 64)
				agatstonScore[agatston] = score
			}
		}

		// 查找動脈分數
		// 從文本中查找動脈分數
		for _, line := range strings.Split(field, ",") {
			if matches := arteryRegex.FindStringSubmatch(line); len(matches) > 0 {
				artery := matches[1]
				score, _ := strconv.ParseFloat(matches[2], 64)
				arteryScores[artery] = score
			}
		}

	}

	// 計算總動脈分數
	totalArteryScore := 0.0
	for _, score := range arteryScores {
		totalArteryScore += score
	}

	maxAgatstonScore := 0.0
	for _, score := range arteryScores {
		if score > maxAgatstonScore {
			maxAgatstonScore = score
		}
	}

	// 比較 Agatston 分數和總動脈分數，選擇較高者
	finalScore := maxAgatstonScore
	if totalArteryScore > maxAgatstonScore {
		finalScore = totalArteryScore
	}
	// 若無法找到分數，則不採用
	// if finalScore == 0 {
	// 	return nil, fmt.Errorf("輸出欄位【%s】數據無效", label)
	// }

	return finalScore, nil
}

func handlePanic(result *Result) {
	if r := recover(); r != nil {
		result.Error = "未知的錯誤"
		result.ErrorCode = "E999"
		result.ErrorDetail = fmt.Sprintf("%v", r)
		result.Data = map[string]interface{}{}
	}
}

func processFile(path string) (result Result) {
	result = initResult()

	defer handlePanic(&result)

	// Check file extension
	fileExt := strings.ToLower(filepath.Ext(path))

	if fileExt == ".xml" {
		// 讀取 XML 檔案
		xmlData, result := readXmlFile(path)
		if result.Error != "" {
			return result
		}

		// 擷取 GetDataResult
		dataResult, result := extractGetDataResult(xmlData)
		if result.Error != "" {
			return result
		}

		// Check if there's data
		dataJSON, result := checkHaveData(dataResult)
		if result.Error != "" {
			return result
		}

		// Extract data from JSON
		return extractdata(dataJSON)

	} else if fileExt == ".txt" {
		// For TXT files, assume it already contains JSON
		dataJSON, result := readTxtFile(path)
		if result.Error != "" {
			return result
		}

		// Extract data directly from JSON
		return extractdata(dataJSON)

	} else {
		result.Error = "Unsupported file format"
		return result
	}
}

// compareCSVWithData compares the CSV data with result.Data
func compareCSVWithData(csvFile string, patientID string, dataMap map[string]interface{}) error {
	file, err := os.Open(csvFile)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Read all rows from the CSV
	rows, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV rows: %v", err)
	}

	// Assuming the first row contains headers
	var modelColumn, valueColumn int
	headers := rows[0]

	// Find columns for "模型用欄位名稱" and "1752487"
	for i, header := range headers {
		if header == "模型用欄位名稱" {
			modelColumn = i
		}
		if header == patientID {
			valueColumn = i
		}
	}

	// Compare CSV values with result.Data
	for _, row := range rows[1:] {
		if len(row) > modelColumn && len(row) > valueColumn {
			csvKey := row[modelColumn]
			csvValue := row[valueColumn]

			if resultValue, exists := dataMap[csvKey]; exists {
				// Convert resultValue to string before comparing
				resultValueStr := fmt.Sprintf("%v", resultValue) // convert to string
				if resultValueStr == csvValue {
					fmt.Printf("Match: Key: %s, Value: %s\n", csvKey, csvValue)
				} else {
					fmt.Printf("Mismatch: Key: %s, CSV Value: %s, Data Value: %s\n", csvKey, csvValue, resultValueStr)
				}
			} else {
				fmt.Printf("Key not found in result.Data: %s\n", csvKey)
			}
		}
	}

	return nil
}

func main() {
	path := filepath.Join("client_rawdata", "2295117_formatted_demo.txt")
	fmt.Printf("路徑: %+v\n", path)
	// var dataMap map[string]interface{}
	result := processFile(path)
	fmt.Println(result.Error)
	fmt.Println(result.ErrorCode)
	fmt.Println(result.ErrorDetail)
	fmt.Println(result.Data)
	for key, value := range result.Data {
		fmt.Printf("%s : %v\n", key, value)
	}

	// // to validate with the demo data
	// csvFile := ".\\HMC_with_demo.csv"
	// patientID := "2295117"
	// err := compareCSVWithData(csvFile, patientID,  result.Data)
	// if err != nil {
	// 	log.Fatalf("Error comparing CSV and data: %v", err)
	// }
}
